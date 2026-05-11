import React, { useRef } from 'react';
import { ReactTabulator } from 'react-tabulator';
import 'tabulator-tables/dist/css/tabulator.min.css';
import './DataTable.css';

const DataTable = ({ data, type, title }) => {
  const tableRef = useRef(null);

  const getColumns = () => {
    if (type === 'students') {
      return [
        { title: '№', field: 'id', width: 50, hozAlign: 'center' },
        { title: 'ФИО', field: 'name', width: 200 },
        { title: 'Группа', field: 'group', width: 100 },
        { title: 'Предмет 1', field: 'subject1', width: 120 },
        { title: 'Предмет 2', field: 'subject2', width: 120 },
        { title: 'Предмет 3', field: 'subject3', width: 120 },
        { title: 'Предмет 4', field: 'subject4', width: 120 }
      ];
    } else if (type === 'attendance') {
      return [
        { title: '№', field: 'id', width: 50, hozAlign: 'center' },
        { title: 'ФИО', field: 'name', width: 200 },
        { title: 'Дата 1', field: 'date1', width: 90, hozAlign: 'center' },
        { title: 'Дата 2', field: 'date2', width: 90, hozAlign: 'center' },
        { title: 'Дата 3', field: 'date3', width: 90, hozAlign: 'center' },
        { title: 'Дата 4', field: 'date4', width: 90, hozAlign: 'center' },
        { title: 'Дата 5', field: 'date5', width: 90, hozAlign: 'center' }
      ];
    }
    return [];
  };

  const options = {
    layout: 'fitColumns',
    pagination: 'local',
    paginationSize: 10,
    movableColumns: true,
    resizableRows: true,
    initialSort: [{ column: 'id', dir: 'asc' }],
    placeholder: 'Нет данных для отображения'
  };

  const tableData = Array.isArray(data) && data.length > 0 ? data : [];

  return (
    <div className="contfor-table">
      <div className="pfp-block-inner">
        <h2>{title}</h2>
        <ReactTabulator
          ref={tableRef}
          columns={getColumns()}
          data={tableData}
          options={options}
        />
      </div>
    </div>
  );
};

export default DataTable;