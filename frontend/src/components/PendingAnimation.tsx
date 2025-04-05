import React, { useState, useEffect, useRef } from 'react';

// Animation frames for a running pixel human
const runnerFrames = [
  `
   O
  /|\\
  / \\
  `,
  `
   O
  /|\\
  | |
  `,
  `
   O
  /|\\
  / \\
  `,
  `
   O
  /|\\
  | |
  `
];

// Ground/desert pattern
const groundPattern = "___.___.___.___.___.___.___.___.___.___.";

const PendingAnimation = () => {
  const [frameIndex, setFrameIndex] = useState(0);
  const [groundPosition, setGroundPosition] = useState(0);
  // const [seconds, setSeconds] = useState(0);
  const requestRef = useRef<number>();
  const previousTimeRef = useRef<number>();
  // const timerRef = useRef<NodeJS.Timeout>();
  
  // Animation frame
  const animate = (time: number) => {
    if (previousTimeRef.current === undefined) {
      previousTimeRef.current = time;
    }
    
    const elapsed = time - (previousTimeRef.current || 0);
    
    // Update animation every 150ms
    if (elapsed > 150) {
      setFrameIndex(prev => (prev + 1) % runnerFrames.length);
      setGroundPosition(prev => (prev + 1) % 10);
      previousTimeRef.current = time;
    }
    
    requestRef.current = requestAnimationFrame(animate);
  };
  
  // Timer
  // useEffect(() => {
  //   timerRef.current = setInterval(() => {
  //     setSeconds(prev => prev + 1);
  //   }, 1000);
    
  //   return () => {
  //     if (timerRef.current) clearInterval(timerRef.current);
  //   };
  // }, []);
  
  // Animation loop
  useEffect(() => {
    requestRef.current = requestAnimationFrame(animate);
    return () => {
      if (requestRef.current) {
        cancelAnimationFrame(requestRef.current);
      }
    };
  }, []);

  // Format time as mm:ss
  // const formatTime = (totalSeconds: number) => {
  //   const minutes = Math.floor(totalSeconds / 60);
  //   const seconds = totalSeconds % 60;
  //   return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
  // };
  
  // Create a scrolling ground effect
  const scrollingGround = groundPattern.substring(groundPosition) + groundPattern.substring(0, groundPosition);
  
  return (
    <div className="flex flex-col items-center my-4">
      <div className="border-2 border-yellow-500 bg-black p-4 w-full max-w-sm overflow-hidden">
        <div className="flex items-center justify-between mb-2">
          <span className="text-yellow-500 text-xs arcade-text">RUNNING TRANSFER...</span>
          <span className="text-yellow-500 text-xs arcade-text blink">PLEASE WAIT</span>
        </div>
        
        <div className="relative flex justify-center py-3 min-h-[60px]">
          {/* Runner */}
          <pre className="text-[#00ff00] whitespace-pre arcade-text leading-tight inline-block transform scale-150">
            {runnerFrames[frameIndex]}
          </pre>
          
          {/* Ground */}
          <div className="text-yellow-500 overflow-hidden whitespace-nowrap absolute bottom-0 left-0 right-0 arcade-text">
            {scrollingGround}
          </div>
        </div>
        
        {/* <div className="flex justify-center mt-3">
          <span className="text-yellow-500 text-xs arcade-text">{formatTime(seconds)}</span>
        </div> */}
      </div>
      
      {/* Custom CSS for blinking text */}
      <style jsx global>{`
        .blink {
          animation: blink-animation 1s steps(2, start) infinite;
        }
        @keyframes blink-animation {
          to {
            visibility: hidden;
          }
        }
      `}</style>
    </div>
  );
};

export default PendingAnimation; 