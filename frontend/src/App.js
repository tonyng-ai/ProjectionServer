import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';

const DEFAULT_CURRENCY = 'USD';
const currencyFormatters = {};
const integerFormatter = new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 });
const decimalFormatter = new Intl.NumberFormat(undefined, { minimumFractionDigits: 0, maximumFractionDigits: 2 });

const getCurrencyFormatter = (currency = DEFAULT_CURRENCY) => {
  const normalized = (currency || DEFAULT_CURRENCY).toUpperCase();
  if (!currencyFormatters[normalized]) {
    try {
      currencyFormatters[normalized] = new Intl.NumberFormat(undefined, { style: 'currency', currency: normalized });
    } catch (err) {
      currencyFormatters[normalized] = new Intl.NumberFormat(undefined, { style: 'currency', currency: DEFAULT_CURRENCY });
    }
  }
  return currencyFormatters[normalized];
};

const parseFormatString = (format) => {
  if (!format) {
    return { type: 'number' };
  }
  const [typePart, detailPart] = format.split(':');
  const type = typePart.trim().toLowerCase();
  const detail = detailPart ? detailPart.trim() : undefined;
  if (type === 'currency') {
    return { type, currency: detail || DEFAULT_CURRENCY };
  }
  if (type === 'count') {
    return { type: 'count' };
  }
  if (type === 'number') {
    return { type: 'number' };
  }
  return { type };
};

const getRowValue = (row, column) => {
  if (!row || !column) {
    return undefined;
  }
  if (Object.prototype.hasOwnProperty.call(row, column)) {
    return row[column];
  }
  const targetKey = Object.keys(row).find((key) => key.toLowerCase() === column.toLowerCase());
  return targetKey ? row[targetKey] : undefined;
};

const formatFieldValue = (value, type) => {
  if (value === null || value === undefined || value === '') {
    return '—';
  }

  switch ((type || '').toLowerCase()) {
    case 'currency': {
      const numeric = Number(value);
      if (Number.isNaN(numeric)) {
        return value;
      }
      return getCurrencyFormatter().format(numeric);
    }
    case 'number': {
      const numeric = Number(value);
      if (Number.isNaN(numeric)) {
        return value;
      }
      return decimalFormatter.format(numeric);
    }
    case 'datetime': {
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) {
        return value;
      }
      return date.toLocaleString();
    }
    case 'date': {
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) {
        return value;
      }
      return date.toLocaleDateString();
    }
    case 'count': {
      const numeric = Number(value);
      if (Number.isNaN(numeric)) {
        return value;
      }
      return integerFormatter.format(numeric);
    }
    default:
      return value;
  }
};

const formatTotalValue = (value, formatMeta) => {
  const safeValue = typeof value === 'number' ? value : parseFloat(value);
  const numeric = Number.isFinite(safeValue) ? safeValue : 0;

  switch ((formatMeta?.type || 'number').toLowerCase()) {
    case 'currency':
      return getCurrencyFormatter(formatMeta.currency).format(numeric);
    case 'count':
      return integerFormatter.format(Math.round(numeric));
    case 'number':
    default:
      return decimalFormatter.format(numeric);
  }
};

function App() {
  const [tables, setTables] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [syncStatus, setSyncStatus] = useState({});
  const [lastRefresh, setLastRefresh] = useState(null);

  const [projections, setProjections] = useState([]);
  const [projectionData, setProjectionData] = useState({});
  const [projectionFilters, setProjectionFilters] = useState({});
  const [projectionSorts, setProjectionSorts] = useState({});
  const [projectionLoading, setProjectionLoading] = useState({});
  const [projectionError, setProjectionError] = useState({});
  const [loadingProjections, setLoadingProjections] = useState(false);

  useEffect(() => {
    fetchStatus();
    fetchProjections();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const fetchStatus = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await axios.get('/api/status');
      setTables(response.data.tables || []);
      setLastRefresh(new Date());
    } catch (err) {
      setError('Failed to fetch status: ' + (err.response?.data?.message || err.message));
      console.error('Error fetching status:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchProjections = async () => {
    try {
      setLoadingProjections(true);
      setError(null);
      const response = await axios.get('/api/projections');
      const configs = response.data.projections || [];
      setProjections(configs);

      const initialFilters = {};
      const initialSorts = {};

      configs.forEach((projection) => {
        initialFilters[projection.id] = {};
        const defaultDirection = projection.default_sort?.direction?.toUpperCase() || 'ASC';
        initialSorts[projection.id] = {
          column: projection.default_sort?.column || '',
          direction: defaultDirection,
        };
      });

      setProjectionFilters(initialFilters);
      setProjectionSorts(initialSorts);

      await Promise.all(
        configs.map((projection) =>
          fetchProjectionData(projection.id, {
            filters: initialFilters[projection.id],
            sort: initialSorts[projection.id],
            projectionOverride: projection,
          })
        )
      );
    } catch (err) {
      setError('Failed to load projections: ' + (err.response?.data?.message || err.message));
      console.error('Error fetching projections:', err);
    } finally {
      setLoadingProjections(false);
    }
  };

  const fetchProjectionData = async (projectionId, options = {}) => {
    const { filters: overrideFilters, sort: overrideSort, projectionOverride } = options;
    const projection = projectionOverride || projections.find((item) => item.id === projectionId);

    if (!projection) {
      return;
    }

    const activeFilters = overrideFilters !== undefined ? overrideFilters : projectionFilters[projectionId] || {};
    const activeSort = overrideSort !== undefined ? overrideSort : projectionSorts[projectionId] || {
      column: projection.default_sort?.column || '',
      direction: (projection.default_sort?.direction || 'ASC').toUpperCase(),
    };

    setProjectionFilters((prev) => ({
      ...prev,
      [projectionId]: activeFilters,
    }));

    setProjectionSorts((prev) => ({
      ...prev,
      [projectionId]: {
        column: activeSort.column || '',
        direction: (activeSort.direction || 'ASC').toUpperCase(),
      },
    }));

    setProjectionLoading((prev) => ({ ...prev, [projectionId]: true }));
    setProjectionError((prev) => ({ ...prev, [projectionId]: null }));

    try {
      const params = new URLSearchParams();

      if (activeSort.column) {
        params.append('sort', activeSort.column);
        params.append('direction', (activeSort.direction || 'ASC').toUpperCase());
      }

      Object.entries(activeFilters).forEach(([filterId, filterValue]) => {
        const filterDef = (projection.filters || []).find((item) => item.id === filterId);
        if (!filterDef) {
          return;
        }

        const filterType = (filterDef.type || '').toLowerCase();

        if (Array.isArray(filterValue)) {
          if (filterValue.length > 0) {
            params.append(`filters[${filterId}]`, filterValue.join(','));
          }
          return;
        }

        if (filterType === 'number') {
          const numeric = parseFloat(filterValue);
          if (!Number.isNaN(numeric)) {
            params.append(`filters[${filterId}]`, numeric);
          }
          return;
        }

        if (filterValue !== undefined && filterValue !== null && `${filterValue}`.trim() !== '') {
          params.append(`filters[${filterId}]`, filterValue);
        }
      });

      const response = await axios.get(`/api/projections/${projectionId}/data`, { params });
      const responseData = response.data || {};

      setProjectionData((prev) => ({
        ...prev,
        [projectionId]: responseData,
      }));

      const metaSort = {
        column: responseData.meta?.sort_column || activeSort.column || '',
        direction: (responseData.meta?.sort_direction || activeSort.direction || 'ASC').toUpperCase(),
      };

      setProjectionSorts((prev) => ({
        ...prev,
        [projectionId]: metaSort,
      }));
    } catch (err) {
      setProjectionError((prev) => ({
        ...prev,
        [projectionId]: err.response?.data?.error || err.message || 'Failed to load data',
      }));
      console.error(`Error fetching projection ${projectionId}:`, err);
    } finally {
      setProjectionLoading((prev) => ({ ...prev, [projectionId]: false }));
    }
  };

  const triggerTableSync = async (tableName, projectionId) => {
    if (!tableName) {
      return;
    }

    try {
      setSyncStatus((prev) => ({ ...prev, [tableName]: 'syncing' }));
      setError(null);

      const response = await axios.post('/api/sync', {
        table_name: tableName,
      });

      if (response.data.success) {
        setSyncStatus((prev) => ({ ...prev, [tableName]: 'success' }));
        setTimeout(() => {
          setSyncStatus((prev) => ({ ...prev, [tableName]: null }));
        }, 3000);

        setTimeout(() => {
          fetchStatus();
          if (projectionId) {
            fetchProjectionData(projectionId);
          }
        }, 1200);
      } else {
        setSyncStatus((prev) => ({ ...prev, [tableName]: 'error' }));
        setError(response.data.message || 'Sync failed');
      }
    } catch (err) {
      setSyncStatus((prev) => ({ ...prev, [tableName]: 'error' }));
      setError('Failed to trigger sync: ' + (err.response?.data?.message || err.message));
      console.error('Error triggering sync:', err);
    }
  };

  const triggerAllSync = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await axios.post('/api/sync', {
        sync_all: true,
      });

      if (response.data.success) {
        setTimeout(() => {
          fetchStatus();
          projections.forEach((projection) => {
            fetchProjectionData(projection.id);
          });
        }, 2000);
      } else {
        setError(response.data.message || 'Sync failed');
      }
    } catch (err) {
      setError('Failed to trigger sync: ' + (err.response?.data?.message || err.message));
      console.error('Error triggering sync:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleProjectionFilterChange = (projection, filter, rawValue, autoFetch = true) => {
    if (!projection || !filter) {
      return;
    }

    const currentFilters = projectionFilters[projection.id] || {};
    const nextFilters = { ...currentFilters };

    if (Array.isArray(rawValue)) {
      if (rawValue.length > 0) {
        nextFilters[filter.id] = rawValue;
      } else {
        delete nextFilters[filter.id];
      }
    } else if (rawValue !== undefined && rawValue !== null && `${rawValue}`.trim() !== '') {
      nextFilters[filter.id] = rawValue;
    } else {
      delete nextFilters[filter.id];
    }

    setProjectionFilters((prev) => ({
      ...prev,
      [projection.id]: nextFilters,
    }));

    if (autoFetch) {
      fetchProjectionData(projection.id, { filters: nextFilters });
    }
  };

  const handleNumberFilterBlur = (projection) => {
    if (!projection) {
      return;
    }
    fetchProjectionData(projection.id);
  };

  const handleProjectionSort = (projection, field) => {
    if (!projection || !field) {
      return;
    }

    const isSortable = field.sortable !== false;
    if (!isSortable) {
      return;
    }

    const currentSort = projectionSorts[projection.id] || {};
    const sameColumn = currentSort.column && field.column && currentSort.column.toLowerCase() === field.column.toLowerCase();
    const nextDirection = sameColumn && currentSort.direction === 'ASC' ? 'DESC' : 'ASC';
    const nextSort = {
      column: field.column,
      direction: nextDirection,
    };

    setProjectionSorts((prev) => ({
      ...prev,
      [projection.id]: nextSort,
    }));

    fetchProjectionData(projection.id, { sort: nextSort });
  };

  const getStatusBadgeClass = (tableName) => {
    const status = syncStatus[tableName];
    if (status === 'syncing') return 'badge-syncing';
    if (status === 'success') return 'badge-success';
    if (status === 'error') return 'badge-error';
    return '';
  };

  const getStatusBadgeText = (tableName) => {
    const status = syncStatus[tableName];
    if (status === 'syncing') return 'Syncing...';
    if (status === 'success') return 'Success!';
    if (status === 'error') return 'Error';
    return '';
  };

  const renderProjectionTable = (projection, rows, fields) => {
    if (!fields || fields.length === 0) {
      return <div className="projection-empty">No fields configured for this projection.</div>;
    }

    const activeSort = projectionSorts[projection.id] || {};

    return (
      <div className="projection-table-wrapper">
        <table className="projection-table">
          <thead>
            <tr>
              {fields.map((field) => {
                const isSortable = field.sortable !== false;
                const isActive =
                  !!activeSort.column &&
                  !!field.column &&
                  activeSort.column.toLowerCase() === field.column.toLowerCase();
                const sortIndicator = !isSortable
                  ? ''
                  : isActive
                  ? activeSort.direction === 'DESC'
                    ? '▼'
                    : '▲'
                  : '↕';

                return (
                  <th
                    key={`${projection.id}-header-${field.column}`}
                    className={isSortable ? 'sortable' : ''}
                    onClick={() => handleProjectionSort(projection, field)}
                  >
                    <span>{field.label || field.column}</span>
                    {isSortable && <span className={`sort-indicator ${isActive ? 'active' : ''}`}>{sortIndicator}</span>}
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr>
                <td className="projection-empty" colSpan={fields.length}>
                  No data available.
                </td>
              </tr>
            ) : (
              rows.map((row, rowIndex) => (
                <tr key={`${projection.id}-row-${rowIndex}`}>
                  {fields.map((field) => {
                    const rawValue = getRowValue(row, field.column);
                    const displayValue = formatFieldValue(rawValue, field.type);
                    const fieldType = (field.type || '').toLowerCase();

                    return (
                      <td key={`${projection.id}-row-${rowIndex}-${field.column}`}>
                        {fieldType === 'badge' ? (
                          <span className={`projection-badge badge-${String(displayValue).toLowerCase().replace(/\s+/g, '-')}`}>
                            {displayValue}
                          </span>
                        ) : (
                          displayValue
                        )}
                      </td>
                    );
                  })}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    );
  };

  const renderGroupedContent = (projection, rows, fields, groupBy, level = 0) => {
    if (!groupBy || groupBy.length === 0) {
      return renderProjectionTable(projection, rows, fields);
    }

    const [currentGroup, ...remainingGroups] = groupBy;
    const grouped = rows.reduce((acc, row) => {
      const value = getRowValue(row, currentGroup);
      const key = value === null || value === undefined || value === '' ? '—' : value;
      if (!acc[key]) {
        acc[key] = [];
      }
      acc[key].push(row);
      return acc;
    }, {});

    return (
      <div className={`projection-group level-${level}`}>
        {Object.entries(grouped).map(([groupValue, groupRows]) => (
          <div className="projection-group-block" key={`${projection.id}-${currentGroup}-${groupValue}-${level}`}>
            <div className="projection-group-header">
              <span className="projection-group-title">
                {currentGroup}: {groupValue}
              </span>
              <span className="projection-group-count">
                {groupRows.length} record{groupRows.length === 1 ? '' : 's'}
              </span>
            </div>
            {renderGroupedContent(projection, groupRows, fields, remainingGroups, level + 1)}
          </div>
        ))}
      </div>
    );
  };

  const renderProjectionTotals = (projection) => {
    if (!projection.totals || projection.totals.length === 0) {
      return null;
    }

    const totals = projectionData[projection.id]?.totals || {};

    const hasTotals = projection.totals.some((totalCfg) => totals[totalCfg.column] !== undefined);
    if (!hasTotals) {
      return null;
    }

    return (
      <div className="projection-card-footer">
        <div className="projection-totals">
          {projection.totals.map((totalCfg) => {
            const totalValue = totals[totalCfg.column];
            const formatMeta = parseFormatString(totalCfg.format);
            const formattedValue = formatTotalValue(totalValue, formatMeta);

            return (
              <div className="projection-total-chip" key={`${projection.id}-total-${totalCfg.column}`}>
                <span className="label">{totalCfg.label || totalCfg.column}</span>
                <span className="value">{formattedValue}</span>
              </div>
            );
          })}
        </div>
      </div>
    );
  };

  const renderProjectionCard = (projection) => {
    const rows = projectionData[projection.id]?.rows || [];
    const isLoading = projectionLoading[projection.id];
    const fields = projection.fields || [];
    const filters = projectionFilters[projection.id] || {};

    return (
      <div className="projection-card" key={projection.id}>
        <div
          className="projection-card-header"
          style={{
            background: projection.header_color || undefined,
            color: projection.header_text_color || undefined,
          }}
        >
          <div className="projection-card-title">
            <h3>{projection.title || projection.target_view}</h3>
            {projection.description && <p>{projection.description}</p>}
          </div>
          <div className="projection-card-actions">
            <button
              className="btn btn-sm btn-secondary"
              onClick={() => fetchProjectionData(projection.id)}
              disabled={isLoading}
            >
              {isLoading ? '⏳ Loading...' : '🔁 Reload Data'}
            </button>
            {projection.sync_table && (
              <button
                className="btn btn-sm btn-primary"
                onClick={() => triggerTableSync(projection.sync_table, projection.id)}
                disabled={syncStatus[projection.sync_table] === 'syncing'}
              >
                {syncStatus[projection.sync_table] === 'syncing' ? '⏳ Syncing...' : '⚡ Trigger Sync'}
              </button>
            )}
          </div>
        </div>
        <div className="projection-card-body">
          {projection.filters && projection.filters.length > 0 && (
            <div className="projection-filters">
              {projection.filters.map((filter) => {
                const filterType = (filter.type || '').toLowerCase();
                const filterValue = filters[filter.id];

                if (filterType === 'select') {
                  return (
                    <div className="projection-filter" key={`${projection.id}-filter-${filter.id}`}>
                      <label>{filter.label || filter.column}</label>
                      <select
                        multiple
                        value={Array.isArray(filterValue) ? filterValue : []}
                        onChange={(event) => {
                          const options = Array.from(event.target.selectedOptions).map((option) => option.value);
                          handleProjectionFilterChange(projection, filter, options, true);
                        }}
                      >
                        {(filter.options || []).map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label || option.value}
                          </option>
                        ))}
                      </select>
                    </div>
                  );
                }

                if (filterType === 'number') {
                  return (
                    <div className="projection-filter" key={`${projection.id}-filter-${filter.id}`}>
                      <label>{filter.label || filter.column}</label>
                      <input
                        type="number"
                        value={filterValue ?? ''}
                        onChange={(event) => handleProjectionFilterChange(projection, filter, event.target.value, false)}
                        onBlur={() => handleNumberFilterBlur(projection)}
                        placeholder="Enter value"
                      />
                    </div>
                  );
                }

                return (
                  <div className="projection-filter" key={`${projection.id}-filter-${filter.id}`}>
                    <label>{filter.label || filter.column}</label>
                    <input
                      type="text"
                      value={filterValue ?? ''}
                      onChange={(event) => handleProjectionFilterChange(projection, filter, event.target.value, false)}
                      onBlur={() => handleNumberFilterBlur(projection)}
                      placeholder="Enter value"
                    />
                  </div>
                );
              })}
            </div>
          )}

          {projectionError[projection.id] && (
            <div className="alert alert-error projection-error">
              <span className="alert-icon">⚠️</span>
              {projectionError[projection.id]}
            </div>
          )}

          {isLoading && rows.length === 0 ? (
            <div className="projection-loading">Loading data...</div>
          ) : projection.group_by && projection.group_by.length > 0 ? (
            renderGroupedContent(projection, rows, fields, projection.group_by)
          ) : (
            renderProjectionTable(projection, rows, fields)
          )}

          {!isLoading && rows.length === 0 && !projectionError[projection.id] && (
            <div className="projection-empty">No records found for the current filters.</div>
          )}

          {isLoading && rows.length > 0 && (
            <div className="projection-loading-overlay">Refreshing…</div>
          )}
        </div>
        {renderProjectionTotals(projection)}
      </div>
    );
  };

  return (
    <div className="App">
      <div className="container">
        <header className="header">
          <h1>🔄 MSSQL → PostgreSQL Sync Service</h1>
          <p className="subtitle">Real-time database synchronization dashboard</p>
        </header>

        {error && (
          <div className="alert alert-error">
            <span className="alert-icon">⚠️</span>
            {error}
          </div>
        )}

        <div className="actions-bar">
          <button
            onClick={fetchStatus}
            className="btn btn-secondary"
            disabled={loading}
          >
            {loading ? '⏳ Loading...' : '🔃 Refresh Status'}
          </button>
          <button
            onClick={triggerAllSync}
            className="btn btn-primary"
            disabled={loading}
          >
            {loading ? '⏳ Syncing...' : '⚡ Sync All Tables'}
          </button>
          <button
            onClick={fetchProjections}
            className="btn btn-secondary"
            disabled={loadingProjections}
          >
            {loadingProjections ? '⏳ Loading...' : '🧭 Reload Projections'}
          </button>
          {lastRefresh && (
            <span className="last-refresh">Last updated: {lastRefresh.toLocaleTimeString()}</span>
          )}
        </div>

        <div className="table-grid">
          {tables.length === 0 ? (
            <div className="empty-state">
              <p>No tables configured for synchronization.</p>
              <p className="empty-state-hint">Check your configuration file.</p>
            </div>
          ) : (
            tables.map((table) => (
              <div key={table.target_table} className="table-card">
                <div className="table-card-header">
                  <h3>{table.target_table}</h3>
                  {syncStatus[table.target_table] && (
                    <span className={`badge ${getStatusBadgeClass(table.target_table)}`}>
                      {getStatusBadgeText(table.target_table)}
                    </span>
                  )}
                </div>

                <div className="table-card-body">
                  <div className="table-info-row">
                    <span className="label">Source:</span>
                    <span className="value">{table.source_table}</span>
                  </div>
                  <div className="table-info-row">
                    <span className="label">Target:</span>
                    <span className="value">{table.target_table}</span>
                  </div>
                  <div className="table-info-row">
                    <span className="label">Refresh Rate:</span>
                    <span className="value">{table.refresh_rate}s</span>
                  </div>

                  <div className="table-features">
                    <span className={`feature-badge ${table.proto_actor_enabled ? 'enabled' : 'disabled'}`}>
                      {table.proto_actor_enabled ? '✓' : '✗'} Auto Sync
                    </span>
                    <span className={`feature-badge ${table.web_api_enabled ? 'enabled' : 'disabled'}`}>
                      {table.web_api_enabled ? '✓' : '✗'} API Trigger
                    </span>
                  </div>
                </div>

                <div className="table-card-footer">
                  {table.web_api_enabled ? (
                    <button
                      onClick={() => triggerTableSync(table.target_table)}
                      className="btn btn-sm btn-primary"
                      disabled={syncStatus[table.target_table] === 'syncing'}
                    >
                      {syncStatus[table.target_table] === 'syncing' ? '⏳ Syncing...' : '🔄 Sync Now'}
                    </button>
                  ) : (
                    <span className="disabled-text">Manual sync disabled</span>
                  )}
                </div>
              </div>
            ))
          )}
        </div>

        <section className="projections-section">
          <div className="projections-header">
            <h2>Projection Views</h2>
            {loadingProjections && <span className="projection-status">Loading projections…</span>}
          </div>

          {projections.length === 0 && !loadingProjections ? (
            <div className="empty-state">
              <p>No projection views configured.</p>
              <p className="empty-state-hint">Add projection definitions to your YAML config.</p>
            </div>
          ) : (
            <div className="projection-grid">
              {projections.map((projection) => renderProjectionCard(projection))}
            </div>
          )}
        </section>

        <footer className="footer">
          <p>Powered by Proto.Actor, Gin, and React.js</p>
        </footer>
      </div>
    </div>
  );
}

export default App;
