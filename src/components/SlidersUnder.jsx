import React, { useState } from 'react';
import './SlidersUnder.css';

const SlidersUnder = () => {
  const [slider1, setSlider1] = useState(50);
  const [slider2, setSlider2] = useState(50);

  return (
    <div className="pfp-sliders-under">
      <div className="pfp-block-inner">
        <div className="slider-container">
          {/* slider content here */}
        </div>
      </div>
    </div>
  );
};

export default SlidersUnder;