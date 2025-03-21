import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { act } from 'react-dom/test-utils';
import { render } from 'test/redux-rtl';

import { FieldType, LogRowContextQueryDirection, LogsSortOrder, MutableDataFrame } from '@grafana/data';
import { dataFrameToLogsModel } from 'app/core/logsModel';

import { LogRowContextModal } from './LogRowContextModal';

const dfBefore = new MutableDataFrame({
  fields: [
    {
      name: 'time',
      type: FieldType.time,
      values: ['2019-04-26T07:28:11.352440161Z', '2019-04-26T09:28:11.352440161Z'],
    },
    {
      name: 'message',
      type: FieldType.string,
      values: ['foo123', 'foo123'],
    },
  ],
});
const dfNow = new MutableDataFrame({
  fields: [
    {
      name: 'time',
      type: FieldType.time,
      values: ['2019-04-26T09:28:11.352440161Z'],
    },
    {
      name: 'message',
      type: FieldType.string,
      values: ['foo123'],
    },
  ],
});
const dfAfter = new MutableDataFrame({
  fields: [
    {
      name: 'time',
      type: FieldType.time,
      values: ['2019-04-26T14:42:50.991981292Z', '2019-04-26T16:28:11.352440161Z'],
    },
    {
      name: 'message',
      type: FieldType.string,
      values: ['foo123', 'bar123'],
    },
  ],
});
const getRowContext = jest.fn().mockImplementation(async (_, options) => {
  if (options.direction === LogRowContextQueryDirection.Forward) {
    return { data: [dfBefore] };
  } else {
    return { data: [dfAfter] };
  }
});
const getRowContextQuery = jest.fn().mockResolvedValue({ datasource: { uid: 'test-uid' } });

const dispatchMock = jest.fn();
jest.mock('app/types', () => ({
  ...jest.requireActual('app/types'),
  useDispatch: () => dispatchMock,
}));

const splitOpen = Symbol('splitOpen');
jest.mock('app/features/explore/state/main', () => ({
  ...jest.requireActual('app/features/explore/state/main'),
  splitOpen: () => splitOpen,
}));

const logs = dataFrameToLogsModel([dfNow]);
const row = logs.rows[0];

const timeZone = 'UTC';

describe('LogRowContextModal', () => {
  const originalScrollIntoView = window.HTMLElement.prototype.scrollIntoView;

  beforeEach(() => {
    window.HTMLElement.prototype.scrollIntoView = jest.fn();
  });
  afterEach(() => {
    window.HTMLElement.prototype.scrollIntoView = originalScrollIntoView;
    jest.clearAllMocks();
  });

  it('should not render modal when it is closed', async () => {
    render(
      <LogRowContextModal row={row} open={false} onClose={() => {}} getRowContext={getRowContext} timeZone={timeZone} />
    );

    await waitFor(() => expect(screen.queryByText('Log context')).not.toBeInTheDocument());
  });

  it('should render modal when it is open', async () => {
    render(
      <LogRowContextModal row={row} open={true} onClose={() => {}} getRowContext={getRowContext} timeZone={timeZone} />
    );

    await waitFor(() => expect(screen.queryByText('Log context')).toBeInTheDocument());
  });

  it('should call getRowContext on open and change of row', async () => {
    render(
      <LogRowContextModal row={row} open={false} onClose={() => {}} getRowContext={getRowContext} timeZone={timeZone} />
    );

    await waitFor(() => expect(getRowContext).not.toHaveBeenCalled());
  });

  it('should call getRowContext on open', async () => {
    render(
      <LogRowContextModal row={row} open={true} onClose={() => {}} getRowContext={getRowContext} timeZone={timeZone} />
    );
    await waitFor(() => expect(getRowContext).toHaveBeenCalledTimes(2));
  });

  it('should render 3 lines containing `foo123`', async () => {
    render(
      <LogRowContextModal
        row={row}
        open={true}
        onClose={() => {}}
        getRowContext={getRowContext}
        timeZone={timeZone}
        logsSortOrder={LogsSortOrder.Descending}
      />
    );
    // there need to be 2 lines with that message. 1 in before, 1 in now, 1 in after
    await waitFor(() => expect(screen.getAllByText('foo123').length).toBe(3));
  });

  it('should call getRowContext when limit changes', async () => {
    render(
      <LogRowContextModal row={row} open={true} onClose={() => {}} getRowContext={getRowContext} timeZone={timeZone} />
    );
    await waitFor(() => expect(getRowContext).toHaveBeenCalledTimes(2));

    const fiftyLinesButton = screen.getByRole('button', {
      name: /50 lines/i,
    });
    await userEvent.click(fiftyLinesButton);
    const twentyLinesButton = screen.getByRole('menuitemradio', {
      name: /20 lines/i,
    });
    await userEvent.click(twentyLinesButton);

    await waitFor(() => expect(getRowContext).toHaveBeenCalledTimes(4));
  });

  it('should call getRowContextQuery when limit changes', async () => {
    render(
      <LogRowContextModal
        row={row}
        open={true}
        onClose={() => {}}
        getRowContext={getRowContext}
        getRowContextQuery={getRowContextQuery}
        timeZone={timeZone}
      />
    );

    // this will call it initially and in the first fetchResults
    await waitFor(() => expect(getRowContextQuery).toHaveBeenCalledTimes(2));

    const tenLinesButton = screen.getByRole('button', {
      name: /50 lines/i,
    });
    await userEvent.click(tenLinesButton);
    const twentyLinesButton = screen.getByRole('menuitemradio', {
      name: /20 lines/i,
    });
    act(() => {
      userEvent.click(twentyLinesButton);
    });

    await waitFor(() => expect(getRowContextQuery).toHaveBeenCalledTimes(3));
  });

  it('should show a split view button', async () => {
    const getRowContextQuery = jest.fn().mockResolvedValue({ datasource: { uid: 'test-uid' } });

    render(
      <LogRowContextModal
        row={row}
        open={true}
        onClose={() => {}}
        getRowContext={getRowContext}
        getRowContextQuery={getRowContextQuery}
        timeZone={timeZone}
      />
    );

    await waitFor(() =>
      expect(
        screen.getByRole('button', {
          name: /open in split view/i,
        })
      ).toBeInTheDocument()
    );
  });

  it('should not show a split view button', async () => {
    render(
      <LogRowContextModal row={row} open={true} onClose={() => {}} getRowContext={getRowContext} timeZone={timeZone} />
    );

    await waitFor(() => {
      expect(
        screen.queryByRole('button', {
          name: /open in split view/i,
        })
      ).not.toBeInTheDocument();
    });
  });

  it('should call getRowContextQuery', async () => {
    const getRowContextQuery = jest.fn().mockResolvedValue({ datasource: { uid: 'test-uid' } });
    render(
      <LogRowContextModal
        row={row}
        open={true}
        onClose={() => {}}
        getRowContext={getRowContext}
        getRowContextQuery={getRowContextQuery}
        timeZone={timeZone}
      />
    );

    await waitFor(() => expect(getRowContextQuery).toHaveBeenCalledTimes(2));
  });

  it('should close modal', async () => {
    const getRowContextQuery = jest.fn().mockResolvedValue({ datasource: { uid: 'test-uid' } });
    const onClose = jest.fn();
    render(
      <LogRowContextModal
        row={row}
        open={true}
        onClose={onClose}
        getRowContext={getRowContext}
        getRowContextQuery={getRowContextQuery}
        timeZone={timeZone}
      />
    );

    const splitViewButton = await screen.findByRole('button', {
      name: /open in split view/i,
    });

    await userEvent.click(splitViewButton);

    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  it('should dispatch splitOpen', async () => {
    const getRowContextQuery = jest.fn().mockResolvedValue({ datasource: { uid: 'test-uid' } });
    const onClose = jest.fn();

    render(
      <LogRowContextModal
        row={row}
        open={true}
        onClose={onClose}
        getRowContext={getRowContext}
        getRowContextQuery={getRowContextQuery}
        timeZone={timeZone}
      />
    );

    const splitViewButton = await screen.findByRole('button', {
      name: /open in split view/i,
    });

    await userEvent.click(splitViewButton);

    await waitFor(() => expect(dispatchMock).toHaveBeenCalledWith(splitOpen));
  });
});
