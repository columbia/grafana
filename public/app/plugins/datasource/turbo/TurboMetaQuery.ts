export function showTables() {
  return 'show_tables';
}

export function getSchema(table?: string) {
  return 'get_schema:' + table;
}
