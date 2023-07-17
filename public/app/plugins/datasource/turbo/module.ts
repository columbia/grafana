import { DataSourcePlugin } from '@grafana/data';
import { SQLQuery } from 'app/features/plugins/sql/types';

import { CheatSheet } from './CheatSheet';
import { QueryEditor } from './QueryEditor';
import { TurboConfigEditor } from './configuration/ConfigurationEditor';
import { TurboDatasource } from './datasource';
import { TurboOptions } from './types';

export const plugin = new DataSourcePlugin<TurboDatasource, SQLQuery, TurboOptions>(TurboDatasource)
  .setQueryEditor(QueryEditor)
  .setQueryEditorHelp(CheatSheet)
  .setConfigEditor(TurboConfigEditor);
