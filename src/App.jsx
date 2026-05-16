// Render student account page
return (
  <div className="contentContainer">
    {/* Кнопка выхода в правом верхнем углу */}
    <div className="logout-top-right">
      <button className="logout-button-top" onClick={handleLogout}>
        Выйти
      </button>
    </div>

    <div className="dashboard-grid">
      <aside className="dashboard-sidebar">
        <ProfileSquare userData={userData} />
      </aside>

      <main className="dashboard-main">
        <ProfileDescription userData={userData} />
        <AttendanceGrid attendanceData={attendanceHeatmapData} maxPerDay={8} />
        <StudentGradesPanel token={token} />
      </main>
    </div>

    <div className="tables-row">
      <DataTable data={sampleStudents} type="students" title="Список студентов" />
      <DataTable data={sampleAttendance} type="attendance" title="Таблица посещаемости" />
    </div>
  </div>
);