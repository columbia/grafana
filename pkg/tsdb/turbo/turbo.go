package turbo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tsdb/sqleng"
)

var logger = log.New("tsdb.turbo")

var (
	errRemoteRequest  = errors.New("remote request error")
	errRemoteResponse = errors.New("remote response error")
)

type DataSourceInfo struct {
	JsonData sqleng.JsonData
	settings backend.DataSourceInstanceSettings
}

type DataPluginConfiguration struct {
	DSInfo            DataSourceInfo
	TimeColumnNames   []string
	MetricColumnTypes []string
	RowLimit          int64
}

type DataSourceHandler struct {
	httpClient             *http.Client
	queryResultTransformer sqleng.SqlQueryResultTransformer
	timeColumnNames        []string
	metricColumnTypes      []string
	log                    log.Logger
	dsInfo                 DataSourceInfo
	rowLimit               int64
}

func NewQueryDataHandler(config DataPluginConfiguration, queryResultTransformer sqleng.SqlQueryResultTransformer, log log.Logger) (*DataSourceHandler, error) {
	logger.Info("Creating Turbo query endpoint")

	opts, err := config.DSInfo.settings.HTTPClientOptions()
	if err != nil {
		return nil, fmt.Errorf("http client options: %w", err)
	}
	cl, err := httpclient.New(opts)
	if err != nil {
		return nil, fmt.Errorf("httpclient new: %w", err)
	}

	queryDataHandler := DataSourceHandler{
		httpClient:             cl,
		queryResultTransformer: queryResultTransformer,
		timeColumnNames:        []string{"time"},
		log:                    log,
		dsInfo:                 config.DSInfo,
		rowLimit:               config.RowLimit,
	}

	if len(config.TimeColumnNames) > 0 {
		queryDataHandler.timeColumnNames = config.TimeColumnNames
	}

	if len(config.MetricColumnTypes) > 0 {
		queryDataHandler.metricColumnTypes = config.MetricColumnTypes
	}
	return &queryDataHandler, nil
}

func (e *DataSourceHandler) TransformQueryError(logger log.Logger, err error) error {
	// OpError is the error type usually returned by functions in the net
	// package. It describes the operation, network type, and address of
	// an error. We log this error rather than return it to the client
	// for security purposes.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		logger.Error("Query error", "err", err)
		return sqleng.ErrConnectionFailed
	}

	return e.queryResultTransformer.TransformQueryError(logger, err)
}

type Service struct {
	im instancemgmt.InstanceManager
}

func ProvideService(cfg *setting.Cfg) *Service {
	s := &Service{}
	s.im = datasource.NewInstanceManager(s.newInstanceSettings(cfg))
	logger.Info("provided service")
	return s
}

func (s *Service) newInstanceSettings(cfg *setting.Cfg) datasource.InstanceFactoryFunc {
	return func(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {

		jsonData := sqleng.JsonData{}
		err := json.Unmarshal(settings.JSONData, &jsonData)
		if err != nil {
			return nil, fmt.Errorf("error reading settings: %w", err)
		}
		dsInfo := DataSourceInfo{
			JsonData: jsonData,
			settings: settings,
		}

		config := DataPluginConfiguration{
			DSInfo:            dsInfo,
			MetricColumnTypes: []string{"UNKNOWN", "TEXT", "VARCHAR", "CHAR"},
			RowLimit:          cfg.DataProxyRowLimit,
		}

		queryResultTransformer := turboQueryResultTransformer{}
		handler, err := NewQueryDataHandler(config, &queryResultTransformer, logger)
		if err != nil {
			logger.Error("Failed connecting to Postgres", "err", err)
			return nil, err
		}
		logger.Debug("Successfully created HTTP client for Turbo")
		return handler, nil
	}
}

func (s *Service) getDSInfo(pluginCtx backend.PluginContext) (*DataSourceHandler, error) {
	i, err := s.im.Get(pluginCtx)
	if err != nil {
		return nil, err
	}
	instance := i.(*DataSourceHandler)
	return instance, nil
}

type DBDataResponse struct {
	dataResponse backend.DataResponse
	refID        string
}

func (s *Service) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	e, err := s.getDSInfo(req.PluginContext)
	if err != nil {
		return nil, err
	}
	result := backend.NewQueryDataResponse()
	ch := make(chan DBDataResponse, len(req.Queries))
	var wg sync.WaitGroup
	// Execute each query in a goroutine and wait for them to finish afterwards
	for _, query := range req.Queries {
		queryjson := sqleng.QueryJson{
			Fill:   false,
			Format: "time_series",
		}
		err := json.Unmarshal(query.JSON, &queryjson)
		if err != nil {
			return nil, fmt.Errorf("error unmarshal query json: %w", err)
		}
		if queryjson.RawSql == "" {
			continue
		}
		wg.Add(1)
		go e.executeQuery(query, &wg, ctx, ch, queryjson, req.PluginContext)
	}

	wg.Wait()

	// Read results from channels
	close(ch)
	result.Responses = make(map[string]backend.DataResponse)
	for queryResult := range ch {
		switch {
		case err == nil:
			result.Responses[queryResult.refID] = queryResult.dataResponse
		case errors.Is(err, context.DeadlineExceeded):
			result.Responses[queryResult.refID] = backend.ErrDataResponse(backend.StatusTimeout, "gateway timeout")
		case errors.Is(err, errRemoteRequest):
			result.Responses[queryResult.refID] = backend.ErrDataResponse(backend.StatusBadGateway, "bad gateway request")
		case errors.Is(err, errRemoteResponse):
			result.Responses[queryResult.refID] = backend.ErrDataResponse(backend.StatusValidationFailed, "bad gateway response")
		default:
			result.Responses[queryResult.refID] = backend.ErrDataResponse(backend.StatusInternal, err.Error())
		}
	}
	return result, nil
}

type resultJson struct {
	Columns     []string   `json:"Columns"`
	ColumnTypes []string   `json:"ColumnTypes"`
	Values      [][]string `json:"Values"`
}

func (e *DataSourceHandler) executeQuery(query backend.DataQuery, wg *sync.WaitGroup, queryContext context.Context,
	ch chan DBDataResponse, queryJson sqleng.QueryJson, pCtx backend.PluginContext) {
	logger.Info("Raw SQL", queryJson.RawSql)
	defer wg.Done()
	queryResult := DBDataResponse{
		dataResponse: backend.DataResponse{},
		refID:        query.RefID,
	}

	logger := e.log.FromContext(queryContext)

	defer func() {
		if r := recover(); r != nil {
			logger.Error("ExecuteQuery panic", "error", r, "stack", log.Stack(1))
			if theErr, ok := r.(error); ok {
				queryResult.dataResponse.Error = theErr
			} else if theErrString, ok := r.(string); ok {
				queryResult.dataResponse.Error = fmt.Errorf(theErrString)
			} else {
				queryResult.dataResponse.Error = fmt.Errorf("unexpected error, see the server log for details")
			}
			ch <- queryResult
		}
	}()

	if queryJson.RawSql == "" {
		panic("Query model property rawSql should not be empty at this point")
	}

	timeRange := query.TimeRange

	errAppendDebug := func(frameErr string, err error, query string) {
		var emptyFrame data.Frame
		emptyFrame.SetMeta(&data.FrameMeta{
			ExecutedQueryString: query,
		})
		queryResult.dataResponse.Error = fmt.Errorf("%s: %w", frameErr, err)
		queryResult.dataResponse.Frames = data.Frames{&emptyFrame}
		ch <- queryResult
	}

	// Create spans for this function.
	// tracing.DefaultTracer() returns the tracer initialized when calling Manage().
	// Refer to OpenTelemetry's Go SDK to know how to customize your spans.
	ctx, span := tracing.DefaultTracer().Start(
		queryContext,
		"query processing",
		trace.WithAttributes(
			attribute.String("query.ref_id", query.RefID),
			attribute.String("query.type", query.QueryType),
			attribute.Int64("query.max_data_points", query.MaxDataPoints),
			attribute.Int64("query.interval_ms", query.Interval.Milliseconds()),
			attribute.Int64("query.time_range.from", query.TimeRange.From.Unix()),
			attribute.Int64("query.time_range.to", query.TimeRange.To.Unix()),
		),
	)
	defer span.End()

	// global substitutions
	interpolatedQuery, err := sqleng.Interpolate(query, timeRange, e.dsInfo.JsonData.TimeInterval, queryJson.RawSql)
	if err != nil {
		errAppendDebug("interpolation failed", e.TransformQueryError(logger, err), interpolatedQuery)
		return
	}

	// Do HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.dsInfo.settings.URL, nil)
	if err != nil {
		errAppendDebug("new request error", e.TransformQueryError(logger, err), interpolatedQuery)
		queryResult.dataResponse.Error = fmt.Errorf("new request with context: %w", err)
	}
	if len(query.JSON) > 0 {
		q := req.URL.Query()
		q.Add("RawSQLQuery", queryJson.RawSql)
		req.URL.RawQuery = q.Encode()
	}
	logger.Info("query JSON", query.JSON)

	httpResp, err := e.httpClient.Do(req)
	switch {
	case err == nil:
		break
	case errors.Is(err, context.DeadlineExceeded):
		queryResult.dataResponse.Error = err
	default:
		queryResult.dataResponse.Error = fmt.Errorf("http client do: %w: %s", errRemoteRequest, err)
	}

	defer func() {
		if err := httpResp.Body.Close(); err != nil {
			logger.Error("query: failed to close response body", "err", err)
		}
	}()
	span.AddEvent("HTTP request done")

	// Make sure the response was successful
	if httpResp.StatusCode != http.StatusOK {
		queryResult.dataResponse.Error = fmt.Errorf("%w: expected 200 response, got %d", errRemoteResponse, httpResp.StatusCode)
	}

	// Decode response
	var rj resultJson
	err = json.NewDecoder(httpResp.Body).Decode(&rj)
	if err != nil {
		queryResult.dataResponse.Error = fmt.Errorf("%w: decode: %s", errRemoteRequest, err)
	}
	logger.Info("result json", rj)
	logger.Info("result json", rj.Columns)
	logger.Info("result json", rj.Columns[0])
	logger.Info("result json", rj.ColumnTypes[0])
	span.AddEvent("JSON response decoded")

	qm, err := e.newProcessCfg(query, queryContext, &rj, interpolatedQuery)
	if err != nil {
		errAppendDebug("failed to get configurations", err, interpolatedQuery)
		return
	}

	// Convert resultJson to dataframe
	// stringConverters := e.queryResultTransformer.GetConverterList()
	frame, err := FrameFromResultJson(rj, e.rowLimit)
	if err != nil {
		errAppendDebug("convert frame from rows error", err, interpolatedQuery)
		return
	}

	if frame.Meta == nil {
		frame.Meta = &data.FrameMeta{}
	}

	frame.Meta.ExecutedQueryString = interpolatedQuery

	// If no rows were returned, no point checking anything else.
	if frame.Rows() == 0 {
		queryResult.dataResponse.Frames = data.Frames{}
		ch <- queryResult
		return
	}

	// if err := convertSQLTimeColumnsToEpochMS(frame, qm); err != nil {
	// 	errAppendDebug("converting time columns failed", err, interpolatedQuery)
	// 	return
	// }

	if qm.Format == dataQueryFormatSeries {
		// time series has to have time column
		if qm.timeIndex == -1 {
			errAppendDebug("db has no time column", errors.New("no time column found"), interpolatedQuery)
			return
		}

		// Make sure to name the time field 'Time' to be backward compatible with Grafana pre-v8.
		frame.Fields[qm.timeIndex].Name = data.TimeSeriesTimeFieldName

		for i := range qm.columnNames {
			if i == qm.timeIndex || i == qm.metricIndex {
				continue
			}

			if t := frame.Fields[i].Type(); t == data.FieldTypeString || t == data.FieldTypeNullableString {
				continue
			}

			// var err error
			// if frame, err = convertSQLValueColumnToFloat(frame, i); err != nil {
			// 	errAppendDebug("convert value to float failed", err, interpolatedQuery)
			// 	return
			// }
		}

		tsSchema := frame.TimeSeriesSchema()
		if tsSchema.Type == data.TimeSeriesTypeLong {
			var err error
			originalData := frame
			frame, err = data.LongToWide(frame, qm.FillMissing)
			if err != nil {
				errAppendDebug("failed to convert long to wide series when converting from dataframe", err, interpolatedQuery)
				return
			}

			// Before 8x, a special metric column was used to name time series. The LongToWide transforms that into a metric label on the value field.
			// But that makes series name have both the value column name AND the metric name. So here we are removing the metric label here and moving it to the
			// field name to get the same naming for the series as pre v8
			if len(originalData.Fields) == 3 {
				for _, field := range frame.Fields {
					if len(field.Labels) == 1 { // 7x only supported one label
						name, ok := field.Labels["metric"]
						if ok {
							field.Name = name
							field.Labels = nil
						}
					}
				}
			}
		}
		// if qm.FillMissing != nil {
		// 	var err error
		// 	frame, err = sqleng.resample(frame, *qm)
		// 	if err != nil {
		// 		logger.Error("Failed to resample dataframe", "err", err)
		// 		frame.AppendNotices(data.Notice{Text: "Failed to resample dataframe", Severity: data.NoticeSeverityWarning})
		// 	}
		// }
	}

	queryResult.dataResponse.Frames = data.Frames{frame}
	ch <- queryResult

	span.AddEvent("Frames created")
}

// CheckHealth pings the connected SQL database
func (s *Service) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {

	e, err := s.getDSInfo(req.PluginContext)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, e.dsInfo.settings.URL, nil)
	if err != nil {
		return newHealthCheckErrorf("could not create request"), nil
	}
	resp, err := e.httpClient.Do(r)
	if err != nil {
		return newHealthCheckErrorf("request error"), nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// log.DefaultLogger.Error("check health: failed to close response body", "err", err.Error())
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return newHealthCheckErrorf("got response code %d", resp.StatusCode), nil
	}
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}

// newHealthCheckErrorf returns a new *backend.CheckHealthResult with its status set to backend.HealthStatusError
// and the specified message, which is formatted with Sprintf.
func newHealthCheckErrorf(format string, args ...interface{}) *backend.CheckHealthResult {
	return &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: fmt.Sprintf(format, args...)}
}

type turboQueryResultTransformer struct{}

func (t *turboQueryResultTransformer) TransformQueryError(_ log.Logger, err error) error {
	return err
}

func (t *turboQueryResultTransformer) GetConverterList() []sqlutil.StringConverter {
	return []sqlutil.StringConverter{}
}

// dataQueryFormat is the type of query.
type dataQueryFormat string

const (
	// dataQueryFormatTable identifies a table query (default).
	dataQueryFormatTable dataQueryFormat = "table"
	// dataQueryFormatSeries identifies a time series query.
	dataQueryFormatSeries dataQueryFormat = "time_series"
)

type dataQueryModel struct {
	InterpolatedQuery string // property not set until after Interpolate()
	Format            dataQueryFormat
	TimeRange         backend.TimeRange
	FillMissing       *data.FillMissing // property not set until after Interpolate()
	Interval          time.Duration
	columnNames       []string
	columnTypes       []string
	timeIndex         int
	timeEndIndex      int
	metricIndex       int
	rj                *resultJson
	metricPrefix      bool
	queryContext      context.Context
}

func (e *DataSourceHandler) newProcessCfg(query backend.DataQuery, queryContext context.Context,
	rj *resultJson, interpolatedQuery string) (*dataQueryModel, error) {

	columnNames := rj.Columns
	columnTypes := rj.ColumnTypes

	qm := &dataQueryModel{
		columnTypes:  columnTypes,
		columnNames:  columnNames,
		rj:           rj,
		timeIndex:    -1,
		timeEndIndex: -1,
		metricIndex:  -1,
		metricPrefix: false,
		queryContext: queryContext,
	}

	queryJson := sqleng.QueryJson{}
	err := json.Unmarshal(query.JSON, &queryJson)
	if err != nil {
		return nil, err
	}

	if queryJson.Fill {
		qm.FillMissing = &data.FillMissing{}
		qm.Interval = time.Duration(queryJson.FillInterval * float64(time.Second))
		switch strings.ToLower(queryJson.FillMode) {
		case "null":
			qm.FillMissing.Mode = data.FillModeNull
		case "previous":
			qm.FillMissing.Mode = data.FillModePrevious
		case "value":
			qm.FillMissing.Mode = data.FillModeValue
			qm.FillMissing.Value = queryJson.FillValue
		default:
		}
	}

	qm.TimeRange.From = query.TimeRange.From.UTC()
	qm.TimeRange.To = query.TimeRange.To.UTC()

	switch queryJson.Format {
	case "time_series":
		qm.Format = dataQueryFormatSeries
	case "table":
		qm.Format = dataQueryFormatTable
	default:
		panic(fmt.Sprintf("Unrecognized query model format: %q", queryJson.Format))
	}

	for i, col := range qm.columnNames {
		for _, tc := range e.timeColumnNames {
			if col == tc {
				qm.timeIndex = i
				break
			}
		}

		if qm.Format == dataQueryFormatTable && col == "timeend" {
			qm.timeEndIndex = i
			continue
		}

		switch col {
		case "metric":
			qm.metricIndex = i
		default:
			if qm.metricIndex == -1 {
				columnType := qm.columnTypes[i]
				for _, mct := range e.metricColumnTypes {
					if columnType == mct {
						qm.metricIndex = i
						continue
					}
				}
			}
		}
	}
	qm.InterpolatedQuery = interpolatedQuery
	return qm, nil
}

func NewFrame(columns []string, columnTypes []string, values [][]string) *data.Frame {

	fields := make(data.Fields, len(columns))

	for i, rowValues := range values {

		switch columnTypes[i] {
		case "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64":
			fields[i] = data.NewField(columns[i], data.Labels{}, castToInt(rowValues))
			break
		case "float32", "float64":
			fields[i] = data.NewField(columns[i], data.Labels{}, castToFloat(rowValues))
			break
		default:
			// Assuming values are strings
			fields[i] = data.NewField(columns[i], data.Labels{}, rowValues)
		}
	}
	return data.NewFrame("", fields...)
}

func castToInt(values []string) []int64 {
	v := make([]int64, len(values))
	for i, value := range values {
		val, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("can't convert to int: %q", val))
		}
		v[i] = val
	}
	return v
}

func castToFloat(values []string) []float64 {
	v := make([]float64, len(values))
	for i, value := range values {
		val, err := strconv.ParseFloat(value, 32)
		if err != nil {
			panic(fmt.Sprintf("can't convert to float: %f", val))
		}
		v[i] = val
	}
	return v
}

func FrameFromResultJson(rj resultJson, rowLimit int64) (*data.Frame, error) {
	types := rj.ColumnTypes
	names := rj.Columns
	values := rj.Values
	frame := NewFrame(names, types, values)
	return frame, nil
}

func SetupFillmode(query *backend.DataQuery, interval time.Duration, fillmode string) error {
	rawQueryProp := make(map[string]interface{})
	queryBytes, err := query.JSON.MarshalJSON()
	if err != nil {
		return err
	}
	err = json.Unmarshal(queryBytes, &rawQueryProp)
	if err != nil {
		return err
	}
	rawQueryProp["fill"] = true
	rawQueryProp["fillInterval"] = interval.Seconds()

	switch fillmode {
	case "NULL":
		rawQueryProp["fillMode"] = "null"
	case "previous":
		rawQueryProp["fillMode"] = "previous"
	default:
		rawQueryProp["fillMode"] = "value"
		floatVal, err := strconv.ParseFloat(fillmode, 64)
		if err != nil {
			return fmt.Errorf("error parsing fill value %v", fillmode)
		}
		rawQueryProp["fillValue"] = floatVal
	}
	query.JSON, err = json.Marshal(rawQueryProp)
	if err != nil {
		return err
	}
	return nil
}
