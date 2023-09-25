import React, { PureComponent } from 'react';

import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { DataSourceHttpSettings } from '@grafana/ui';

import { TurboOptions } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<TurboOptions> {}

interface State {}

export class TurboConfigEditor extends PureComponent<Props, State> {
  render() {
    const { options } = this.props;
    return (
      <div className="gf-form-group">
        <div className="gf-form">
          <DataSourceHttpSettings
            defaultUrl="http://127.0.0.1:10000"
            dataSourceConfig={options}
            onChange={this.props.onOptionsChange}
          />
        </div>
      </div>
    );
  }
}
