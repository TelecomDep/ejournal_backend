import React from 'react';
import './InfoCard.css';

const InfoCard = ({ title, value, color, position }) => {
  const positions = {
    1: { left: '213px', top: '879px' },
    2: { left: '590px', top: '879px' },
    3: { left: '972px', top: '879px' }
  };

  const style = {
    ...positions[position],
    backgroundColor: color
  };

  return (
    <div className="info-card" style={positions[position]}>
      <div className="pfp-block-inner">
        <h3>{title}</h3>
        <div className="card-value">{value}</div>
      </div>
    </div>
  );
};

export default InfoCard;