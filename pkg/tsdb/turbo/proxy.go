package turbo

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"net"
	"time"

	iproxy "github.com/grafana/grafana/pkg/infra/proxy"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tsdb/sqleng"
	"github.com/grafana/grafana/pkg/util"
	"github.com/lib/pq"
	"golang.org/x/net/proxy"
	"xorm.io/core"
)

// createTurboProxyDriver creates and registers a new sql driver that uses a turbo connector and updates the dialer to
// route connections through the secure socks proxy
func createTurboProxyDriver(settings *setting.SecureSocksDSProxySettings, cnnstr string) (string, error) {
	sqleng.XormDriverMu.Lock()
	defer sqleng.XormDriverMu.Unlock()

	// create a unique driver per connection string
	hash, err := util.Md5SumString(cnnstr)
	if err != nil {
		return "", err
	}
	driverName := "turbo-proxy-" + hash

	// only register the driver once
	if core.QueryDriver(driverName) == nil {
		connector, err := pq.NewConnector(cnnstr)
		if err != nil {
			return "", err
		}

		driver, err := newTurboProxyDriver(settings, connector)
		if err != nil {
			return "", err
		}

		sql.Register(driverName, driver)
		core.RegisterDriver(driverName, driver)
	}
	return driverName, nil
}

// turboProxyDriver is a regular turbo driver with an updated dialer.
// This is done because there is no way to save a dialer to the turbo driver in xorm
type turboProxyDriver struct {
	c *pq.Connector
}

var _ driver.DriverContext = (*turboProxyDriver)(nil)
var _ core.Driver = (*turboProxyDriver)(nil)

// newTurboProxyDriver updates the dialer for a turbo connector with a dialer that proxys connections through the secure socks proxy
// and returns a new turbo driver to register
func newTurboProxyDriver(cfg *setting.SecureSocksDSProxySettings, connector *pq.Connector) (*turboProxyDriver, error) {
	dialer, err := iproxy.NewSecureSocksProxyContextDialer(cfg)
	if err != nil {
		return nil, err
	}

	// update the turbo dialer with the proxy dialer
	connector.Dialer(&turboProxyDialer{d: dialer})

	return &turboProxyDriver{connector}, nil
}

// turboProxyDialer implements the turbo dialer using a proxy dialer, as their functions differ slightly
type turboProxyDialer struct {
	d proxy.Dialer
}

// Dial uses the normal proxy dial function with the updated dialer
func (p *turboProxyDialer) Dial(network, addr string) (c net.Conn, err error) {
	return p.d.Dial(network, addr)
}

// DialTimeout uses the normal turbo dial timeout function with the updated dialer
func (p *turboProxyDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return p.d.(proxy.ContextDialer).DialContext(ctx, network, address)
}

// Parse uses the xorm turbo dialect for the driver (this has to be implemented to register the driver with xorm)
func (d *turboProxyDriver) Parse(a string, b string) (*core.Uri, error) {
	sqleng.XormDriverMu.RLock()
	defer sqleng.XormDriverMu.RUnlock()

	return core.QueryDriver("turbo").Parse(a, b)
}

// OpenConnector returns the normal turbo connector that has the updated dialer context
func (d *turboProxyDriver) OpenConnector(name string) (driver.Connector, error) {
	return d.c, nil
}

// Open uses the connector with the updated dialer to open a new connection
func (d *turboProxyDriver) Open(dsn string) (driver.Conn, error) {
	return d.c.Connect(context.Background())
}
