'use client';

interface Runner {
  address: string;
  score: string;
}

interface ColorClasses {
  container: string;
  headerText: string;
  rowText: string;
  headerBorder: string;
  rowBorder: string;
  hoverBg: string;
  glowBg: string;
  glowBgHover: string;
}

interface LeaderboardTableProps {
  chainName: string;
  colorClasses: ColorClasses;
  runners: Runner[];
}

// LeaderboardTable component to display a single chain's leaderboard
function LeaderboardTable({ chainName, colorClasses, runners }: LeaderboardTableProps) {
  // Helper function to ensure 'x' is lowercase in addresses
  const formatAddress = (address: string) => {
    return address.replace('0X', '0x');
  };

  return (
    <div className={`arcade-container ${colorClasses.container} relative group`}>
      <div className={`absolute inset-0 ${colorClasses.glowBg} blur-sm group-hover:${colorClasses.glowBgHover} transition-all duration-300`} />
      <div className="relative">
        <h3 className={`arcade-text text-lg mb-4 ${colorClasses.headerText} text-center`}>
          {chainName}
        </h3>
        
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className={`border-b ${colorClasses.headerBorder}`}>
                <th className={`py-2 px-4 text-left arcade-text text-xs ${colorClasses.headerText}`}>RANK</th>
                <th className={`py-2 px-4 text-left arcade-text text-xs ${colorClasses.headerText}`}>SPEEDRUNNER</th>
                <th className={`py-2 px-4 text-right arcade-text text-xs ${colorClasses.headerText}`}>SCORE</th>
              </tr>
            </thead>
            <tbody>
              {runners.map((runner, index) => (
                <tr key={index} className={`border-b ${colorClasses.rowBorder} hover:${colorClasses.hoverBg}`}>
                  <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText}`}>#{index + 1}</td>
                  <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText}`}>{formatAddress(runner.address)}</td>
                  <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} text-right`}>{runner.score}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

export default function Leaderboard() {
  // Sample placeholder data for all chains
  const placeholderRunners = [
    { address: '0x0000000000000000000000000000000000000000', score: '0:00' },
    { address: '0x0000000000000000000000000000000000000000', score: '0:00' },
    { address: '0x0000000000000000000000000000000000000000', score: '0:00' },
  ];

  // Chain configurations with fixed class names
  const chains = [
    { 
      name: 'BASE',
      colorClasses: {
        container: 'border-blue-500',
        headerText: 'text-blue-500',
        rowText: 'text-blue-300',
        headerBorder: 'border-blue-500/40',
        rowBorder: 'border-blue-500/20',
        hoverBg: 'hover:bg-blue-500/10',
        glowBg: 'bg-blue-500/10',
        glowBgHover: 'bg-blue-500/20'
      }
    },
    { 
      name: 'ARBITRUM',
      colorClasses: {
        container: 'border-indigo-700',
        headerText: 'text-indigo-700',
        rowText: 'text-indigo-300',
        headerBorder: 'border-indigo-700/40',
        rowBorder: 'border-indigo-700/20',
        hoverBg: 'hover:bg-indigo-700/10',
        glowBg: 'bg-indigo-700/10',
        glowBgHover: 'bg-indigo-700/20'
      }
    },
    { 
      name: 'BNB CHAIN',
      colorClasses: {
        container: 'border-yellow-500',
        headerText: 'text-yellow-500',
        rowText: 'text-yellow-300',
        headerBorder: 'border-yellow-500/40',
        rowBorder: 'border-yellow-500/20',
        hoverBg: 'hover:bg-yellow-500/10',
        glowBg: 'bg-yellow-500/10',
        glowBgHover: 'bg-yellow-500/20'
      }
    },
    { 
      name: 'POLYGON',
      colorClasses: {
        container: 'border-purple-600',
        headerText: 'text-purple-600',
        rowText: 'text-purple-300',
        headerBorder: 'border-purple-600/40',
        rowBorder: 'border-purple-600/20',
        hoverBg: 'hover:bg-purple-600/10',
        glowBg: 'bg-purple-600/10',
        glowBgHover: 'bg-purple-600/20'
      }
    },
    { 
      name: 'AVALANCHE',
      colorClasses: {
        container: 'border-red-600',
        headerText: 'text-red-600',
        rowText: 'text-red-300',
        headerBorder: 'border-red-600/40',
        rowBorder: 'border-red-600/20',
        hoverBg: 'hover:bg-red-600/10',
        glowBg: 'bg-red-600/10',
        glowBgHover: 'bg-red-600/20'
      }
    },
    { 
      name: 'ETHEREUM',
      colorClasses: {
        container: 'border-gray-500',
        headerText: 'text-gray-500',
        rowText: 'text-gray-300',
        headerBorder: 'border-gray-500/40',
        rowBorder: 'border-gray-500/20',
        hoverBg: 'hover:bg-gray-500/10',
        glowBg: 'bg-gray-500/10',
        glowBgHover: 'bg-gray-500/20'
      }
    }
  ];

  return (
    <main className="flex min-h-screen flex-col items-center p-8 relative overflow-hidden">
      {/* Retro grid background */}
      <div className="fixed inset-0 bg-[linear-gradient(transparent_1px,_transparent_1px),_linear-gradient(90deg,_transparent_1px,_transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_80%_50%_at_50%_0%,_#000_70%,_transparent_100%)] opacity-20" />
      
      {/* Neon glow effects */}
      <div className="fixed inset-0 bg-[radial-gradient(circle_at_50%_50%,_rgba(255,255,0,0.1)_0%,_transparent_50%)]" />
      
      <div className="z-10 max-w-5xl w-full relative">
        <div className="text-center mb-8">
          <h1 className="arcade-text text-3xl text-primary-500 relative mb-4">
            <span className="absolute inset-0 blur-sm opacity-50">LEADERBOARD</span>
            LEADERBOARD
          </h1>
          <div className="arcade-text text-xl text-yellow-500 mb-4 animate-pulse">
            COMING SOON...
          </div>
          <p className="arcade-text text-sm text-primary-300 relative">
            BECOME THE FASTEST SPEEDRUNNER AND EARN LEGENDARY REWARDS
          </p>
        </div>

        <div className="mt-12 space-y-8">
          {chains.map((chain, index) => (
            <LeaderboardTable
              key={index}
              chainName={chain.name}
              colorClasses={chain.colorClasses}
              runners={placeholderRunners}
            />
          ))}
        </div>
      </div>
    </main>
  );
} 