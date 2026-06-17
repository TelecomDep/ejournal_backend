import React, { useState } from 'react';
import './AttendanceGrid.css';

const AttendanceGrid = ({ attendanceData = [], maxPerDay = 8 }) => {
  const [selectedDay, setSelectedDay] = useState(null);

  // Генерация 32 ячеек в сетке 4x8
  const generateCells = () => {
    const cells = [];
    const today = new Date();
    
    // Создаем 32 ячейки, соответствующие последним 32 дням
    for (let i = 31; i >= 0; i--) {
      const date = new Date();
      date.setDate(today.getDate() - i);
      const dateStr = date.toISOString().split('T')[0];
      const attendance = attendanceData.find(a => a.date === dateStr);
      const count = attendance ? attendance.count : 0;
      
      cells.push({
        id: 31 - i,
        date: dateStr,
        count: count,
        max: maxPerDay,
        level: count === 0 ? -1 : Math.min(4, Math.ceil((count / maxPerDay) * 4))
      });
    }
    
    return cells;
  };

  const cells = generateCells();

  const getCellColor = (count, max) => {
    if (count === 0) {
      return '#FF4B2F'; // Красный для 0 посещений
    }
    if (count === max) {
      return '#6B8ED4'; // Максимальная яркость для max из max
    }
    // Плавный переход от бледного к яркому
    const ratio = count / max;
    const intensity = 0.3 + ratio * 0.7;
    return `rgba(107, 142, 212, ${intensity})`;
  };

  const getTooltipText = (cell) => {
    if (cell.count === 0) {
      return `${new Date(cell.date).toLocaleDateString('ru-RU')}\n0 из ${cell.max} посещений`;
    }
    return `${new Date(cell.date).toLocaleDateString('ru-RU')}\n${cell.count} из ${cell.max} посещений`;
  };

  const groupCellPositions = [
    { group: 3, col: 0, row: 0, pos: [90, 6], transform: 'rotate(180deg)' },
    { group: 3, col: 1, row: 0, pos: [60, 6], transform: 'rotate(180deg)' },
    { group: 3, col: 2, row: 0, pos: [30, 6], transform: 'rotate(180deg)' },
    { group: 3, col: 3, row: 0, pos: [0, 6], transform: 'rotate(180deg)' },
    { group: 3, col: 0, row: 1, pos: [0, 64], transform: '' },
    { group: 3, col: 1, row: 1, pos: [30, 35], transform: '' },
    { group: 3, col: 2, row: 1, pos: [60, 64], transform: '' },
    { group: 3, col: 3, row: 1, pos: [90, 64], transform: '' },
  ];

  return (
    <div className="attendance-grid">
      <div className="attendance-grid-header">
        <span className="grid-title">Посещаемость</span>
        <span className="grid-stats">
          {attendanceData.reduce((sum, a) => sum + a.count, 0)} всего посещений
        </span>
      </div>
      
      <div className="attendance-grid-container">
        <div className="grid-labels">
          <div className="month-labels">
            <span>Мар</span>
            <span>Апр</span>
            <span>Май</span>
            <span>Июн</span>
            <span>Июл</span>
            <span>Авг</span>
            <span>Сен</span>
            <span>Окт</span>
          </div>
          <div className="weekday-labels">
            <span>Пн</span>
            <span>Ср</span>
            <span>Пт</span>
          </div>
        </div>
        
        <div className="cells-wrapper">
          {/* Group 3 */}
          <div className="cell-group" style={{ left: '0px', top: '6px', width: '120px', height: '204px' }}>
            {cells.slice(0, 8).map((cell, idx) => {
              const pos = groupCellPositions[idx];
              return (
                <div
                  key={cell.id}
                  className="attendance-cell"
                  style={{
                    position: 'absolute',
                    left: `${pos.pos[0]}px`,
                    top: `${pos.pos[1]}px`,
                    width: '30px',
                    height: '30px',
                    backgroundColor: getCellColor(cell.count, cell.max),
                    borderRadius: '5px',
                    transform: pos.transform || '',
                    cursor: 'pointer'
                  }}
                  onMouseEnter={(e) => {
                    setSelectedDay({ ...cell, x: e.clientX, y: e.clientY });
                  }}
                  onMouseLeave={() => setSelectedDay(null)}
                >
                  {cell.count > 0 && cell.count === cell.max && (
                    <span className="cell-star">★</span>
                  )}
                </div>
              );
            })}
          </div>
          
          {/* Group 4 */}
          <div className="cell-group" style={{ left: '120px', top: '6px', width: '120px', height: '204px' }}>
            {cells.slice(8, 16).map((cell, idx) => {
              const pos = groupCellPositions[idx];
              return (
                <div
                  key={cell.id}
                  className="attendance-cell"
                  style={{
                    position: 'absolute',
                    left: `${pos.pos[0]}px`,
                    top: `${pos.pos[1]}px`,
                    width: '30px',
                    height: '30px',
                    backgroundColor: getCellColor(cell.count, cell.max),
                    borderRadius: '5px',
                    transform: pos.transform || '',
                    cursor: 'pointer'
                  }}
                  onMouseEnter={(e) => {
                    setSelectedDay({ ...cell, x: e.clientX, y: e.clientY });
                  }}
                  onMouseLeave={() => setSelectedDay(null)}
                >
                  {cell.count > 0 && cell.count === cell.max && (
                    <span className="cell-star">★</span>
                  )}
                </div>
              );
            })}
          </div>
          
          {/* Group 5 */}
          <div className="cell-group" style={{ left: '240px', top: '6px', width: '120px', height: '204px' }}>
            {cells.slice(16, 24).map((cell, idx) => {
              const pos = groupCellPositions[idx];
              return (
                <div
                  key={cell.id}
                  className="attendance-cell"
                  style={{
                    position: 'absolute',
                    left: `${pos.pos[0]}px`,
                    top: `${pos.pos[1]}px`,
                    width: '30px',
                    height: '30px',
                    backgroundColor: getCellColor(cell.count, cell.max),
                    borderRadius: '5px',
                    transform: pos.transform || '',
                    cursor: 'pointer'
                  }}
                  onMouseEnter={(e) => {
                    setSelectedDay({ ...cell, x: e.clientX, y: e.clientY });
                  }}
                  onMouseLeave={() => setSelectedDay(null)}
                >
                  {cell.count > 0 && cell.count === cell.max && (
                    <span className="cell-star">★</span>
                  )}
                </div>
              );
            })}
          </div>
          
          {/* Group 6 */}
          <div className="cell-group" style={{ left: '360px', top: '6px', width: '120px', height: '204px' }}>
            {cells.slice(24, 32).map((cell, idx) => {
              const pos = groupCellPositions[idx];
              return (
                <div
                  key={cell.id}
                  className="attendance-cell"
                  style={{
                    position: 'absolute',
                    left: `${pos.pos[0]}px`,
                    top: `${pos.pos[1]}px`,
                    width: '30px',
                    height: '30px',
                    backgroundColor: getCellColor(cell.count, cell.max),
                    borderRadius: '5px',
                    transform: pos.transform || '',
                    cursor: 'pointer'
                  }}
                  onMouseEnter={(e) => {
                    setSelectedDay({ ...cell, x: e.clientX, y: e.clientY });
                  }}
                  onMouseLeave={() => setSelectedDay(null)}
                >
                  {cell.count > 0 && cell.count === cell.max && (
                    <span className="cell-star">★</span>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </div>
      
      {selectedDay && (
        <div 
          className="attendance-tooltip"
          style={{
            position: 'fixed',
            left: selectedDay.x + 15,
            top: selectedDay.y - 40,
            zIndex: 1000
          }}
        >
          <strong>{new Date(selectedDay.date).toLocaleDateString('ru-RU')}</strong>
          <span>
            {selectedDay.count === 0 
              ? `0 из ${selectedDay.max} посещений`
              : `${selectedDay.count} из ${selectedDay.max} посещений`
            }
          </span>
          <div className="tooltip-bar">
            <div 
              className="tooltip-fill"
              style={{ width: `${(selectedDay.count / selectedDay.max) * 100}%` }}
            />
          </div>
        </div>
      )}
      
      <div className="legend">
        <span>Меньше</span>
        <div className="legend-colors">
          <div className="legend-color" style={{ backgroundColor: 'rgba(156, 160, 147, 0.15)' }} />
          <div className="legend-color" style={{ backgroundColor: 'rgba(107, 142, 212, 0.3)' }} />
          <div className="legend-color" style={{ backgroundColor: 'rgba(107, 142, 212, 0.55)' }} />
          <div className="legend-color" style={{ backgroundColor: 'rgba(107, 142, 212, 0.8)' }} />
          <div className="legend-color" style={{ backgroundColor: '#6B8ED4' }} />
        </div>
        <span>Больше</span>
        <div className="legend-red">
          <div className="legend-color" style={{ backgroundColor: '#FF4B2F' }} />
          <span>0 из n</span>
        </div>
      </div>
    </div>
  );
};

export default AttendanceGrid;
