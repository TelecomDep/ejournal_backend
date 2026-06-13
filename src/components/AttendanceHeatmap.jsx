import React, { useState, useEffect } from 'react';
import './AttendanceHeatmap.css';

// Локальная дата в формате YYYY-MM-DD без перевода в UTC,
// чтобы ячейки совпадали с датами посещений из БД (часовой пояс сервера).
const toLocalDateStr = (date) => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const AttendanceHeatmap = ({ attendanceData = [], year = new Date().getFullYear() }) => {
  const [selectedCell, setSelectedCell] = useState(null);

  // Генерация данных для тепловой карты (52 недели × 7 дней)
  const generateHeatmapData = () => {
    const today = new Date();
    const startDate = new Date(year, 0, 1);
    // Начинаем с воскресенья для правильного выравнивания
    const firstDayOffset = startDate.getDay();
    startDate.setDate(startDate.getDate() - firstDayOffset);
    
    const endDate = new Date(year, 11, 31);
    const lastDayOffset = 6 - endDate.getDay();
    endDate.setDate(endDate.getDate() + lastDayOffset);
    
    const weeks = [];
    const currentDate = new Date(startDate);
    
    while (currentDate <= endDate) {
      const week = [];
      for (let i = 0; i < 7; i++) {
        const date = new Date(currentDate);
        const dateStr = toLocalDateStr(date);
        const attendance = attendanceData.find(a => a.date === dateStr);
        week.push({
          date: dateStr,
          count: attendance ? attendance.count : 0,
          level: getIntensityLevel(attendance ? attendance.count : 0),
          isCurrentMonth: date.getMonth() === new Date().getMonth(),
          weekday: i
        });
        currentDate.setDate(currentDate.getDate() + 1);
      }
      weeks.push(week);
    }
    
    return weeks;
  };
  
  // Определение уровня интенсивности на основе количества посещений
  const getIntensityLevel = (count) => {
    if (count === 0) return 0;
    if (count <= 2) return 1;
    if (count <= 5) return 2;
    if (count <= 8) return 3;
    return 4;
  };
  
  // Получение цвета ячейки по уровню
  const getCellColor = (level, isSelected = false) => {
    if (isSelected) return '#7C3AED';
    const colors = {
      0: 'rgba(156, 160, 147, 0.15)',
      1: 'rgba(107, 142, 212, 0.3)',
      2: 'rgba(107, 142, 212, 0.55)',
      3: 'rgba(107, 142, 212, 0.8)',
      4: '#6B8ED4'
    };
    return colors[level];
  };
  
  const heatmapData = generateHeatmapData();
  
  // Месяцы для отображения
  const months = ['Янв', 'Фев', 'Мар', 'Апр', 'Май', 'Июн', 'Июл', 'Авг', 'Сен', 'Окт', 'Ноя', 'Дек'];
  
  // Дни недели
  const weekdays = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
  
  // Получение позиции месяца
  const getMonthPosition = () => {
    const positions = [];
    let currentMonth = -1;
    
    heatmapData.forEach((week, weekIndex) => {
      const firstDayOfWeek = week[0];
      const month = new Date(firstDayOfWeek.date).getMonth();
      
      if (month !== currentMonth) {
        positions.push({
          month: months[month],
          weekIndex,
          monthIndex: month
        });
        currentMonth = month;
      }
    });
    
    return positions;
  };
  
  const monthPositions = getMonthPosition();
  
  // Статистика
  const totalAttendance = attendanceData.reduce((sum, item) => sum + item.count, 0);
  const activeDays = attendanceData.filter(item => item.count > 0).length;
  const bestDay = attendanceData.reduce((best, item) => item.count > best.count ? item : best, { count: 0 });
  
  const handleCellClick = (cell) => {
    setSelectedCell(cell);
  };
  
  return (
    <div className="attendance-heatmap">
      <div className="heatmap-header">
        <div className="heatmap-title-section">
          <h3 className="heatmap-title">Посещаемость</h3>
          <span className="heatmap-stats">
            {totalAttendance} посещений · {activeDays} дней
          </span>
        </div>
        <div className="heatmap-legend">
          <span className="legend-label">Меньше</span>
          <div className="legend-colors">
            <div className="legend-cell" style={{ backgroundColor: 'rgba(156, 160, 147, 0.15)' }}></div>
            <div className="legend-cell" style={{ backgroundColor: 'rgba(107, 142, 212, 0.3)' }}></div>
            <div className="legend-cell" style={{ backgroundColor: 'rgba(107, 142, 212, 0.55)' }}></div>
            <div className="legend-cell" style={{ backgroundColor: 'rgba(107, 142, 212, 0.8)' }}></div>
            <div className="legend-cell" style={{ backgroundColor: '#6B8ED4' }}></div>
          </div>
          <span className="legend-label">Больше</span>
        </div>
      </div>
      
      <div className="heatmap-container">
        <div className="weekdays-labels">
          <div></div>
          {weekdays.map((day, index) => (
            <div key={index} className="weekday-label">{day}</div>
          ))}
        </div>
        
        <div className="heatmap-wrapper">
          <div className="months-row">
            {monthPositions.map((month, idx) => (
              <div 
                key={idx} 
                className="month-label"
                style={{ left: `${month.weekIndex * 16}px` }}
              >
                {month.month}
              </div>
            ))}
          </div>
          
          <div className="heatmap-grid">
            {heatmapData.map((week, weekIndex) => (
              <div key={weekIndex} className="heatmap-week">
                {week.map((day, dayIndex) => (
                  <div
                    key={`${weekIndex}-${dayIndex}`}
                    className={`heatmap-cell ${selectedCell?.date === day.date ? 'selected' : ''}`}
                    style={{ 
                      backgroundColor: getCellColor(day.level, selectedCell?.date === day.date)
                    }}
                    onClick={() => handleCellClick(day)}
                    title={`${day.date}: ${day.count} посещений`}
                  >
                    {day.count > 0 && day.level >= 3 && (
                      <span className="cell-indicator"></span>
                    )}
                  </div>
                ))}
              </div>
            ))}
          </div>
        </div>
      </div>
      
      {selectedCell && (
        <div className="heatmap-tooltip">
          <strong>{new Date(selectedCell.date).toLocaleDateString('ru-RU', {
            day: 'numeric',
            month: 'long',
            year: 'numeric'
          })}</strong>
          <span>{selectedCell.count} посещений</span>
        </div>
      )}
      
      <div className="heatmap-footer">
        <div className="heatmap-badge">
          <span className="badge-icon">📊</span>
          <span>Лучший день: {bestDay.date ? new Date(bestDay.date).toLocaleDateString('ru-RU') : '—'} ({bestDay.count} посещ.)</span>
        </div>
        <div className="heatmap-controls">
          <button 
            className="year-btn"
            onClick={() => window.location.reload()}
          >
            {year}
          </button>
        </div>
      </div>
    </div>
  );
};

export default AttendanceHeatmap;