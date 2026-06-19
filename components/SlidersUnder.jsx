import React, { useState } from 'react';
import './SlidersUnder.css';

const SlidersUnder = () => {
  const [slider1, setSlider1] = useState(50);
  const [slider2, setSlider2] = useState(75);

  return (
    <div className="pfp-sliders-under">
      <div className="pfp-block-inner">
        <div className="slider-container">
          <label>
            <span className="slider-title">Параметр 1</span>
            <span className="slider-value">{slider1}%</span>
          </label>
          <input
            type="range"
            min="0"
            max="100"
            value={slider1}
            onChange={(e) => setSlider1(Number(e.target.value))}
            className="custom-slider"
          />
        </div>
        
        <div className="slider-container">
          <label>
            <span className="slider-title">Параметр 2</span>
            <span className="slider-value">{slider2}%</span>
          </label>
          <input
            type="range"
            min="0"
            max="100"
            value={slider2}
            onChange={(e) => setSlider2(Number(e.target.value))}
            className="custom-slider"
          />
        </div>
      </div>
    </div>
  );
};

export default SlidersUnder;