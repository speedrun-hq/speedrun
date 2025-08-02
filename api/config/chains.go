package config

import "fmt"

const (
	ZetachainMainnetChainID = 7000
)

const (
	ethereumMainnetChainID  = 1
	bscMainnetChainID       = 56
	polygonMainnetChainID   = 137
	arbitrumMainnetChainID  = 42161
	baseMainnetChainID      = 8453
	avalancheMainnetChainID = 43114

	ethereumSepoliaChainID  = 11155111
	bscTestnetChainID       = 97
	polygonAmoyChainID      = 80002
	arbitrumSepoliaChainID  = 421614
	baseSepoliaChainID      = 84532
	zetachainTestnetChainID = 7001
	avalancheFujiChainID    = 43113

	ethereumName  = "ETHEREUM"
	bscName       = "BSC"
	polygonName   = "POLYGON"
	arbitrumName  = "ARBITRUM"
	baseName      = "BASE"
	zetachainName = "ZETACHAIN"
	avalancheName = "AVALANCHE"

	mainnetDefaultChains = "42161,8453,137,1,43114,56,7000"
	testnetDefaultChains = "421614,84532,80002,11155111,43113,97,7001"
)

var intentAddressByChain = map[uint64]string{
	ethereumMainnetChainID:  "0x951AB2A5417a51eB5810aC44BC1fC716995C1CAB",
	bscMainnetChainID:       "0x68282fa70a32E52711d437b6c5984B714Eec3ED0",
	polygonMainnetChainID:   "0x4017717c550E4B6E61048D412a718D6A8078d264",
	arbitrumMainnetChainID:  "0xD6B0E2a8D115cCA2823c5F80F8416644F3970dD2",
	baseMainnetChainID:      "0x999fce149FD078DCFaa2C681e060e00F528552f4",
	ZetachainMainnetChainID: "0x986e2db1aF08688dD3C9311016026daD15969e09",
	avalancheMainnetChainID: "0x9a22A7d337aF1801BEEcDBE7f4f04BbD09F9E5bb",
}

// chainNameFromID returns the chain name based on the chain ID
func chainNameFromID(chainID uint64) (string, error) {
	switch chainID {
	case arbitrumMainnetChainID, arbitrumSepoliaChainID:
		return arbitrumName, nil
	case baseMainnetChainID, baseSepoliaChainID:
		return baseName, nil
	case ZetachainMainnetChainID, zetachainTestnetChainID:
		return zetachainName, nil
	case polygonMainnetChainID, polygonAmoyChainID:
		return polygonName, nil
	case ethereumMainnetChainID, ethereumSepoliaChainID:
		return ethereumName, nil
	case bscMainnetChainID, bscTestnetChainID:
		return bscName, nil
	case avalancheMainnetChainID, avalancheFujiChainID:
		return avalancheName, nil
	}
	return "", fmt.Errorf("unsupported chain ID: %d", chainID)
}
