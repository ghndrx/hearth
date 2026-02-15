import React from 'react';

interface TypingIndicatorProps {
  className?: string;
  show?: boolean;
  username?: string;
}

const TypingIndicator: React.FC<TypingIndicatorProps> = ({ 
  className = '', 
  show = true,
  username = 'User'
}) => {
  if (!show) return null;

  return (
    <div className={`typing-indicator ${className}`} style={styles.container}>
      <span style={styles.text}>{username} is typing</span>
      <span style={styles.dots}>
        <span style={{ ...styles.dot, animationDelay: '0s' }}>.</span>
        <span style={{ ...styles.dot, animationDelay: '0.2s' }}>.</span>
        <span style={{ ...styles.dot, animationDelay: '0.4s' }}>.</span>
      </span>
      <style>{keyframes}</style>
    </div>
  );
};

const keyframes = `
  @keyframes typingDot {
    0%, 20% {
      opacity: 0;
    }
    50% {
      opacity: 1;
    }
    100% {
      opacity: 0;
    }
  }
`;

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    display: 'inline-flex',
    alignItems: 'center',
    padding: '8px 12px',
    fontSize: '14px',
    color: '#666',
  },
  text: {
    marginRight: '2px',
  },
  dots: {
    display: 'inline-flex',
    marginLeft: '2px',
  },
  dot: {
    animation: 'typingDot 1.4s infinite',
    fontSize: '14px',
    fontWeight: 'bold',
  },
};

export default TypingIndicator;
