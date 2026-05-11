import React from 'react';
import './InfoCard.css';

const InfoCard = ({ title, value, color, onClick }) => {
  return (
    <div className="info-card" onClick={onClick}>
      <div className="pfp-block-inner">
        <div className="card-icon" style={{ background: color }}>
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
            <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <path d="M2 17L12 22L22 17" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <path d="M2 12L12 17L22 12" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </div>
        <h3 className="card-title">{title}</h3>
        <div className="card-value">{value}</div>
        <div className="card-footer">
          <span className="card-detail">Подробнее</span>
        </div>
      </div>
    </div>
  );
};

export default InfoCard;