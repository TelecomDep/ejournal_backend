import React, { useState } from 'react';
import './TeacherPanels.css';

const TeacherGrades = ({ data, sendMessage, teacherId }) => {
  const [gradeForm, setGradeForm] = useState({
    studentId: '',
    subjectId: '1',
    score: '',
    maxScore: '10',
    itemType: 'laboratory',
    title: '',
    comment: ''
  });

  const handleGradeSubmit = () => {
    sendMessage({
      action: 'saveGrade',
      data: {
        teacher_id: teacherId,
        ...gradeForm,
        score: parseInt(gradeForm.score),
        max_score: parseInt(gradeForm.maxScore),
        student_id: parseInt(gradeForm.studentId),
        subject_id: parseInt(gradeForm.subjectId)
      }
    });
  };

  const gradeItems = [
    { id: 1, title: 'Лаб. работа 1', maxScore: 10, type: 'laboratory', deadline: '15.02.2024' },
    { id: 2, title: 'Контрольная 1', maxScore: 20, type: 'test', deadline: '01.03.2024' },
    { id: 3, title: 'Проект', maxScore: 30, type: 'project', deadline: '20.03.2024' },
    { id: 4, title: 'Экзамен', maxScore: 40, type: 'exam', deadline: '15.04.2024' }
  ];

  return (
    <div className="teacher-panel">
      <h2 className="panel-title">Управление оценками</h2>
      
      <div className="panel-grid">
        <div className="panel-card">
          <div className="pfp-block-inner-dark">
            <h3>Выставить оценку</h3>
            
            <div className="form-row">
              <div className="form-group">
                <label>ID студента</label>
                <input 
                  type="number" 
                  value={gradeForm.studentId}
                  onChange={(e) => setGradeForm({...gradeForm, studentId: e.target.value})}
                  className="dark-input"
                />
              </div>
              
              <div className="form-group">
                <label>ID предмета</label>
                <input 
                  type="number" 
                  value={gradeForm.subjectId}
                  onChange={(e) => setGradeForm({...gradeForm, subjectId: e.target.value})}
                  className="dark-input"
                />
              </div>
            </div>
            
            <div className="form-group">
              <label>Контрольная точка</label>
              <select 
                className="dark-input"
                onChange={(e) => {
                  const selected = gradeItems.find(item => item.id === parseInt(e.target.value));
                  if (selected) {
                    setGradeForm({
                      ...gradeForm,
                      title: selected.title,
                      maxScore: selected.maxScore.toString(),
                      itemType: selected.type
                    });
                  }
                }}
              >
                <option value="">Выберите работу</option>
                {gradeItems.map(item => (
                  <option key={item.id} value={item.id}>
                    {item.title} (макс. {item.maxScore} б.)
                  </option>
                ))}
              </select>
            </div>
            
            <div className="form-row">
              <div className="form-group">
                <label>Баллы</label>
                <input 
                  type="number" 
                  value={gradeForm.score}
                  onChange={(e) => setGradeForm({...gradeForm, score: e.target.value})}
                  className="dark-input"
                  max={gradeForm.maxScore}
                />
              </div>
              
              <div className="form-group">
                <label>Максимум</label>
                <input 
                  type="number" 
                  value={gradeForm.maxScore}
                  onChange={(e) => setGradeForm({...gradeForm, maxScore: e.target.value})}
                  className="dark-input"
                  readOnly
                />
              </div>
            </div>
            
            <div className="form-group">
              <label>Комментарий</label>
              <textarea 
                value={gradeForm.comment}
                onChange={(e) => setGradeForm({...gradeForm, comment: e.target.value})}
                className="dark-input"
                rows="3"
                placeholder="Комментарий к оценке"
              />
            </div>
            
            <button onClick={handleGradeSubmit} className="action-btn">
              Сохранить оценку
            </button>
          </div>
        </div>
        
        <div className="panel-card">
          <div className="pfp-block-inner-dark">
            <h3>Контрольные точки</h3>
            <div className="grade-items-list">
              {gradeItems.map(item => (
                <div key={item.id} className="grade-item-card">
                  <div className="grade-item-header">
                    <span className="grade-item-title">{item.title}</span>
                    <span className={`grade-item-type ${item.type}`}>
                      {item.type === 'laboratory' && 'Лаб.'}
                      {item.type === 'test' && 'Тест'}
                      {item.type === 'project' && 'Проект'}
                      {item.type === 'exam' && 'Экзамен'}
                    </span>
                  </div>
                  <div className="grade-item-details">
                    <span>Макс. балл: {item.maxScore}</span>
                    <span>Срок: {item.deadline}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default TeacherGrades;