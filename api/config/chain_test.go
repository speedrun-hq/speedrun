package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainNameFromID(t *testing.T) {
	tests := []struct {
		chainID uint64
		name    string
		wantErr bool
	}{
		{arbitrumMainnetChainID, arbitrumName, false},
		{baseMainnetChainID, baseName, false},
		{zetachainMainnetChainID, zetachainName, false},
		{polygonMainnetChainID, polygonName, false},
		{ethereumMainnetChainID, ethereumName, false},
		{bscMainnetChainID, bscName, false},
		{avalancheMainnetChainID, avalancheName, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("chainID=%d", tt.chainID), func(t *testing.T) {
			got, err := chainNameFromID(tt.chainID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.name, got)
			}
		})
	}
}
