'use client';

import { useState, useEffect } from 'react';
import { apiService } from '@/services/api';
import { Runner, Intent } from '@/types';

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
  // Helper function to truncate address for display
  const truncateAddress = (address: string | undefined | null) => {
    if (!address) return 'Unknown';
    address = address.replace('0X', '0x');
    if (address.length <= 10) return address;
    return `${address.substring(0, 6)}...${address.substring(address.length - 4)}`;
  };

  return (
    <div className={`arcade-container ${colorClasses.container} relative group h-full`}>
      <div className={`absolute inset-0 ${colorClasses.glowBg} blur-sm group-hover:${colorClasses.glowBgHover} transition-all duration-300`} />
      <div className="relative">
        <h3 className={`arcade-text text-lg mb-4 ${colorClasses.headerText} text-center`}>
          {chainName}
        </h3>
        
        <div className="overflow-x-auto">
          <table className="w-full table-fixed">
            <thead>
              <tr className={`border-b ${colorClasses.headerBorder}`}>
                <th className={`py-2 px-4 text-left arcade-text text-xs ${colorClasses.headerText} w-[80px]`}>RANK</th>
                <th className={`py-2 px-4 text-left arcade-text text-xs ${colorClasses.headerText} w-[180px]`}>SPEEDRUNNER</th>
                <th className={`py-2 px-4 text-right arcade-text text-xs ${colorClasses.headerText} w-[100px]`}>RUNS</th>
                <th className={`py-2 px-4 text-right arcade-text text-xs ${colorClasses.headerText}`}>VOLUME</th>
              </tr>
            </thead>
            <tbody>
              {runners.length > 0 ? (
                Array.from({ length: 5 }, (_, index) => {
                  const runner = runners[index];
                  if (runner) {
                    return (
                      <tr key={index} className={`border-b ${colorClasses.rowBorder} hover:${colorClasses.hoverBg}`}>
                        <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} truncate`}>#{index + 1}</td>
                        <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} truncate`}>{truncateAddress(runner.address)}</td>
                        <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} text-right`}>{runner.total_transfers}</td>
                        <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} text-right whitespace-nowrap`}>{runner.total_volume}</td>
                      </tr>
                    );
                  } else {
                    return (
                      <tr key={index} className={`border-b ${colorClasses.rowBorder} opacity-30`}>
                        <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} truncate`}>#{index + 1}</td>
                        <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} truncate`}>-</td>
                        <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} text-right`}>-</td>
                        <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} text-right whitespace-nowrap`}>-</td>
                      </tr>
                    );
                  }
                })
              ) : (
                Array.from({ length: 5 }, (_, index) => (
                  <tr key={index} className={`border-b ${colorClasses.rowBorder} opacity-30`}>
                    <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} truncate`}>#{index + 1}</td>
                    <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} truncate`}>-</td>
                    <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} text-right`}>-</td>
                    <td className={`py-3 px-4 arcade-text text-xs ${colorClasses.rowText} text-right whitespace-nowrap`}>-</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

export default function Leaderboard() {
  const [leaderboardData, setLeaderboardData] = useState<{ [key: string]: Runner[] }>({});
  const [loading, setLoading] = useState(true);
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  useEffect(() => {
    const fetchLeaderboardData = async () => {
      console.log('Starting fetchLeaderboardData');
      try {
        setLoading(true);
        setErrors({});

        // Fetch all intents
        console.log('About to fetch intents from API...');
        const response = await apiService.listIntents();
        console.log('Raw API Response:', JSON.stringify(response, null, 2));

        // Check if response is empty or undefined
        if (!response) {
          console.warn('API response is empty or undefined');
          setLeaderboardData({});
          return;
        }

        const intents: Intent[] = Array.isArray(response) ? response : [];
        
        // Log the first intent to see its structure
        if (intents.length > 0) {
          console.log('First intent structure:', {
            keys: Object.keys(intents[0]),
            fullIntent: intents[0]
          });
        }

        // Check if we have any intents
        if (intents.length === 0) {
          console.warn('No intents found in the response');
          setLeaderboardData({});
          return;
        }
        
        // Process intents into leaderboard data
        const chainIds = {
          'BASE': 8453,
          'ARBITRUM': 42161,
          'BNB CHAIN': 56,
          'POLYGON': 137,
          'AVALANCHE': 43114,
          'ETHEREUM': 1
        };

        const data: { [key: string]: Runner[] } = {};
        const newErrors: { [key: string]: string } = {};

        // Process intents for each chain
        Object.entries(chainIds).forEach(([chainName, chainId]) => {
          try {
            // Filter intents for this chain
            const chainIntents = intents.filter((intent: Intent) => {
              console.log(`Checking intent for ${chainName}:`, {
                intent_chain: intent.destination_chain,
                chain_id: chainId,
                recipient: intent.recipient
              });
              
              const destChainId = typeof intent.destination_chain === 'string' 
                ? parseInt(intent.destination_chain) 
                : intent.destination_chain;
              
              return destChainId === chainId && intent.recipient && intent.status === 'settled';
            });
            
            console.log(`${chainName} filtered intents:`, chainIntents);
            
            if (chainIntents.length === 0) {
              console.log(`No intents found for ${chainName}`);
              data[chainName] = [];
              return;
            }
            
            // Group intents by recipient address
            const runnerMap = new Map<string, Runner>();
            
            chainIntents.forEach((intent: Intent) => {
              const address = intent.recipient;
              if (!address) {
                console.log('Skipping intent with no recipient:', intent);
                return;
              }
              
              console.log(`Processing intent for ${address}:`, intent);
              
              const existingRunner = runnerMap.get(address);
              
              // Helper function to convert amount based on source chain
              const getHumanReadableAmount = (amount: string, sourceChain?: string | number) => {
                const decimals = sourceChain === '56' || sourceChain === 56 ? 18 : 6;
                return parseFloat(amount) / Math.pow(10, decimals);
              };

              if (existingRunner) {
                // Update existing runner
                existingRunner.total_transfers += 1;
                // Convert based on source chain decimals
                const currentVolume = parseFloat(existingRunner.total_volume.replace(' USDC', '').replace(',', ''));
                const newAmount = getHumanReadableAmount(intent.amount, intent.source_chain);
                existingRunner.total_volume = `${(currentVolume + newAmount).toLocaleString()} USDC`;
                
                // Update fastest time if this transfer was faster
                const transferTime = new Date(intent.updated_at).getTime() - new Date(intent.created_at).getTime();
                const currentFastest = parseFloat(existingRunner.fastest_time.replace('s', ''));
                if (transferTime < currentFastest * 1000) {
                  existingRunner.fastest_time = `${(transferTime / 1000).toFixed(1)}s`;
                }
                
                // Update last transfer if this one is more recent
                if (new Date(intent.updated_at) > new Date(existingRunner.last_transfer)) {
                  existingRunner.last_transfer = intent.updated_at;
                }
                
                // Update score based on transfers only
                existingRunner.score = (existingRunner.total_transfers * 10).toString();

                console.log(`Updated runner ${address}:`, existingRunner);
              } else {
                // Create new runner
                const transferTime = new Date(intent.updated_at).getTime() - new Date(intent.created_at).getTime();
                const newRunner = {
                  address,
                  score: "10", // Initial score for first transfer
                  total_transfers: 1,
                  total_volume: `${getHumanReadableAmount(intent.amount, intent.source_chain).toLocaleString()} USDC`,
                  fastest_time: `${(transferTime / 1000).toFixed(1)}s`,
                  last_transfer: intent.updated_at,
                  average_time: `${(transferTime / 1000).toFixed(1)}s`
                };
                runnerMap.set(address, newRunner);
                console.log(`Created new runner ${address}:`, newRunner);
              }
            });
            
            // Convert map to array and sort by score
            data[chainName] = Array.from(runnerMap.values())
              .sort((a, b) => parseInt(b.score) - parseInt(a.score))
              .slice(0, 5); // Keep top 5 runners

            console.log(`Final leaderboard for ${chainName}:`, data[chainName]);
              
          } catch (err) {
            console.error(`Error processing data for ${chainName}:`, err);
            data[chainName] = [];
            newErrors[chainName] = err instanceof Error ? err.message : 'Failed to process data';
          }
        });

        console.log('Final leaderboard data:', data);
        setLeaderboardData(data);
        setErrors(newErrors);
      } catch (err) {
        console.error('Error in fetchLeaderboardData:', err);
        setErrors({ global: err instanceof Error ? err.message : 'Failed to fetch leaderboard data' });
        setLeaderboardData({});
      } finally {
        console.log('fetchLeaderboardData completed');
        setLoading(false);
      }
    };

    console.log('Setting up leaderboard data fetch');
    fetchLeaderboardData();
  }, []);

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
      
      <div className="z-10 max-w-6xl w-full relative">
        <div className="text-center mb-8">
          <h1 className="arcade-text text-3xl text-primary-500 relative mb-4">
            <span className="absolute inset-0 blur-sm opacity-50">LEADERBOARD</span>
            LEADERBOARD
          </h1>
          {loading ? (
            <div className="arcade-text text-xl text-yellow-500 mb-4 animate-pulse">
              LOADING...
            </div>
          ) : errors.global ? (
            <div className="arcade-text text-xl text-red-500 mb-4">
              {errors.global}
            </div>
          ) : (
            <p className="arcade-text text-sm text-primary-300 relative">
              BECOME THE FASTEST SPEEDRUNNER AND EARN LEGENDARY REWARDS
            </p>
          )}
        </div>

        <div className="mt-12">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {chains.map((chain, index) => (
              <div key={index} className="relative">
                {errors[chain.name] && (
                  <div className={`absolute -top-6 left-0 right-0 arcade-text text-xs ${chain.colorClasses.headerText} text-center`}>
                    ERROR: {errors[chain.name]}
                  </div>
                )}
                <LeaderboardTable
                  chainName={chain.name}
                  colorClasses={chain.colorClasses}
                  runners={leaderboardData[chain.name] || []}
                />
              </div>
            ))}
          </div>
        </div>
      </div>
    </main>
  );
} 