import React, { useEffect, useState } from 'react';
import api from '../services/api';
import AttendanceHeatmap from '../components/AttendanceHeatmap';

const AttendancePage = ({ token }) => {
  const [items, setItems] = useState([]);

  useEffect(() => {
    let cancelled = false;
    api.getStudentAttendanceHeatmap(token, new Date().getFullYear())
      .then((data) => {
        if (!cancelled) {
          setItems(data.items || []);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setItems([]);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [token]);

  return <AttendanceHeatmap attendanceData={items} year={new Date().getFullYear()} />;
};

export default AttendancePage;
