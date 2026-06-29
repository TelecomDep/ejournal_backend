import React from 'react';

const SibLogo = ({ size = 40, withWordmark = false, className = '' }) => {
  const mark = (
    <img
      src={`${process.env.PUBLIC_URL}/sibguti-logo.png`}
      width={size}
      height={size}
      alt="СибГУТИ"
      className="sib-logo-mark"
    />
  );

  if (!withWordmark) {
    return mark;
  }

  return (
    <span className={`sib-logo ${className}`}>
      {mark}
      <span className="sib-logo-word">СибГУТИ</span>
    </span>
  );
};

export default SibLogo;
