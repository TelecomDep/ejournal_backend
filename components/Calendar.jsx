import React, { useState } from 'react';
import ReactCalendar from 'react-calendar';
import 'react-calendar/dist/Calendar.css';
import './Calendar.css';

const Calendar = () => {
  const [date, setDate] = useState(new Date());

  return (
    <div className="pfp-calendar">
      <div className="pfp-block-inner">
        <h3 className="calendar-title">Календарь</h3>
        <ReactCalendar
          onChange={setDate}
          value={date}
          className="react-calendar"
          tileClassName={({ date, view }) => {
            if (view === 'month') {
              const today = new Date();
              if (date.toDateString() === today.toDateString()) {
                return 'today-tile';
              }
            }
            return null;
          }}
        />
      </div>
    </div>
  );
};

export default Calendar;