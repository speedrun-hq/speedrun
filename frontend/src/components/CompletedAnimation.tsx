import React, { useState, useEffect, useRef } from 'react';

// Animation frames for the jumping celebration
const celebrationFrames = [
  `
   \\O/
    |
   / \\
  `,
  `
    O
   /|\\
   / \\
  `,
  `
   \\O/
    |
   / \\
  `,
  `
    O
   \\|/
   / \\
  `
];

// Ground pattern
const groundPattern = "___.___.___.___.___.___.___.___.___.___.";

interface CompletedAnimationProps {
}

const CompletedAnimation: React.FC<CompletedAnimationProps> = ({ }) => {
  const [frameIndex, setFrameIndex] = useState(0);
  const requestRef = useRef<number>();
  const previousTimeRef = useRef<number>();
  
  // Animation frame
  const animate = (time: number) => {
    if (previousTimeRef.current === undefined) {
      previousTimeRef.current = time;
    }
    
    const elapsed = time - (previousTimeRef.current || 0);
    
    // Update animation every 300ms (victory dance)
    if (elapsed > 300) {
      setFrameIndex(prev => (prev + 1) % celebrationFrames.length);
      previousTimeRef.current = time;
    }
    
    requestRef.current = requestAnimationFrame(animate);
  };
  
  // Animation loop
  useEffect(() => {
    requestRef.current = requestAnimationFrame(animate);
    return () => {
      if (requestRef.current) {
        cancelAnimationFrame(requestRef.current);
      }
    };
  }, []);
  
  return (
    <div className="flex flex-col items-center my-4 w-full">
      <div className="border-2 border-green-500 bg-black p-4 w-full max-w-sm overflow-hidden">
        <div className="flex justify-between mb-2">
          <div className="flex flex-col items-center">
            <span className="text-green-500 text-xs arcade-text">TRANSFER</span>
            <span className="text-green-500 text-xs arcade-text">DONE</span>
          </div>
          <div className="flex flex-col items-center">
            <span className="text-green-500 text-xs arcade-text">LEVEL</span>
            <span className="text-green-500 text-xs arcade-text">COMPLETE</span>
          </div>
        </div>
        
        <div className="relative flex justify-center py-3 min-h-[60px]">
          {/* Celebrating Figure */}
          <pre className="text-[#00ff00] whitespace-pre arcade-text leading-tight inline-block transform scale-150">
            {celebrationFrames[frameIndex]}
          </pre>
          
          {/* Ground */}
          <div className="text-green-500 overflow-hidden whitespace-nowrap absolute bottom-0 left-0 right-0 arcade-text">
            {groundPattern}
          </div>
        </div>
        
        {/* <div className="flex justify-center mt-3">
          <span className="text-green-500 text-xs arcade-text">TIME: {time}</span>
        </div> */}
      </div>
    </div>
  );
};

export default CompletedAnimation; 