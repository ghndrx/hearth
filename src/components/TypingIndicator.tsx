import React from 'react';

interface TypingIndicatorProps {
  className?: string;
  show?: boolean;
}

const TypingIndicator: React.FC<TypingIndicatorProps> = ({ className = '', show = true }) => {
  if (!show) return null;

  return (
    <div className={`typing-indicator ${className}`}>
      <span className="dots">
        <span />
        <span />
        <span />
      </span>
    </div>
  );
};

export default TypingIndicator;