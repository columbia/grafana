import React from 'react';

import { QueryEditorProps } from '@grafana/data';
import { SqlQueryEditor } from 'app/features/plugins/sql/components/QueryEditor';
import { SQLOptions, SQLQuery } from 'app/features/plugins/sql/types';

import { TurboDatasource } from './datasource';

const queryHeaderProps = { isDatasetSelectorHidden: true };

export function QueryEditor(props: QueryEditorProps<TurboDatasource, SQLQuery, SQLOptions>) {
  return <SqlQueryEditor {...props} queryHeaderProps={queryHeaderProps} />;
}
