import { render, screen } from '@testing-library/react';
import React from 'react';
import { AutoSizerProps } from 'react-virtualized-auto-sizer';
import { TestProvider } from 'test/helpers/TestProvider';

import { DataSourceApi, LoadingState, CoreApp, createTheme, EventBusSrv } from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import { ExploreId } from 'app/types';

import { Explore, Props } from './Explore';
import { scanStopAction } from './state/query';
import { createEmptyQueryResponse } from './state/utils';

const resizeWindow = (x: number, y: number) => {
  global.innerWidth = x;
  global.innerHeight = y;
  global.dispatchEvent(new Event('resize'));
};

const makeEmptyQueryResponse = (loadingState: LoadingState) => {
  const baseEmptyResponse = createEmptyQueryResponse();

  baseEmptyResponse.request = {
    requestId: '1',
    intervalMs: 0,
    interval: '1s',
    panelId: 1,
    range: baseEmptyResponse.timeRange,
    scopedVars: {
      apps: {
        value: 'value',
        text: 'text',
      },
    },
    targets: [
      {
        refId: 'A',
      },
    ],
    timezone: 'UTC',
    app: CoreApp.Explore,
    startTime: 0,
  };

  baseEmptyResponse.state = loadingState;

  return baseEmptyResponse;
};

const dummyProps: Props = {
  logsResult: undefined,
  changeSize: jest.fn(),
  datasourceInstance: {
    meta: {
      metrics: true,
      logs: true,
    },
    components: {
      QueryEditorHelp: {},
    },
  } as DataSourceApi,
  datasourceMissing: false,
  exploreId: ExploreId.left,
  loading: false,
  modifyQueries: jest.fn(),
  scanStart: jest.fn(),
  scanStopAction: scanStopAction,
  setQueries: jest.fn(),
  queryKeys: [],
  queries: [],
  isLive: false,
  syncedTimes: false,
  updateTimeRange: jest.fn(),
  makeAbsoluteTime: jest.fn(),
  graphResult: [],
  absoluteRange: {
    from: 0,
    to: 0,
  },
  timeZone: 'UTC',
  queryResponse: makeEmptyQueryResponse(LoadingState.NotStarted),
  addQueryRow: jest.fn(),
  theme: createTheme(),
  showMetrics: true,
  showLogs: true,
  showTable: true,
  showTrace: true,
  showNodeGraph: true,
  showFlameGraph: true,
  splitOpen: () => {},
  splitted: false,
  isFromCompactUrl: false,
  eventBus: new EventBusSrv(),
  showRawPrometheus: false,
  showLogsSample: false,
  logsSample: { enabled: false },
  setSupplementaryQueryEnabled: jest.fn(),
};

jest.mock('@grafana/runtime/src/services/dataSourceSrv', () => {
  return {
    getDataSourceSrv: () => ({
      get: () => Promise.resolve({}),
      getList: () => [],
      getInstanceSettings: () => {},
    }),
  };
});

jest.mock('app/core/core', () => ({
  contextSrv: {
    hasAccess: () => true,
  },
}));

// for the AutoSizer component to have a width
jest.mock('react-virtualized-auto-sizer', () => {
  return ({ children }: AutoSizerProps) => children({ height: 1, width: 1 });
});

const setup = (overrideProps?: Partial<Props>) => {
  const exploreProps = { ...dummyProps, ...overrideProps };

  return render(
    <TestProvider>
      <Explore {...exploreProps} />
    </TestProvider>
  );
};

describe('Explore', () => {
  it('should not render no data with not started loading state', async () => {
    setup();

    // Wait for the Explore component to render
    await screen.findByTestId(selectors.components.DataSourcePicker.container);

    expect(screen.queryByTestId('explore-no-data')).not.toBeInTheDocument();
  });

  it('should render no data with done loading state', async () => {
    const queryResp = makeEmptyQueryResponse(LoadingState.Done);
    setup({ queryResponse: queryResp });

    // Wait for the Explore component to render
    await screen.findByTestId(selectors.components.DataSourcePicker.container);

    expect(screen.getByTestId('explore-no-data')).toBeInTheDocument();
  });

  describe('On small screens', () => {
    const windowWidth = global.innerWidth,
      windowHeight = global.innerHeight;

    beforeAll(() => {
      resizeWindow(500, 500);
    });

    afterAll(() => {
      resizeWindow(windowWidth, windowHeight);
    });

    it('should render data source picker', async () => {
      setup();

      const dataSourcePicker = await screen.findByTestId(selectors.components.DataSourcePicker.container);

      expect(dataSourcePicker).toBeInTheDocument();
    });
  });
});
