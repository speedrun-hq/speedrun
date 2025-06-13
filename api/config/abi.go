package config

// IntentInitiatedEventABI is the ABI for the IntentInitiated and IntentInitiatedWithCall events
const IntentInitiatedEventABI = `[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "bytes32",
				"name": "intentId",
				"type": "bytes32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "asset",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "targetChain",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "bytes",
				"name": "receiver",
				"type": "bytes"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "tip",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "salt",
				"type": "uint256"
			}
		],
		"name": "IntentInitiated",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"internalType": "bytes32",
				"name": "intentId",
				"type": "bytes32"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "asset",
				"type": "address"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "targetChain",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "bytes",
				"name": "receiver",
				"type": "bytes"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "tip",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "uint256",
				"name": "salt",
				"type": "uint256"
			},
			{
				"indexed": false,
				"internalType": "bytes",
				"name": "data",
				"type": "bytes"
			}
		],
		"name": "IntentInitiatedWithCall",
		"type": "event"
	}
]`

// IntentFulfilledEventABI is the ABI for the IntentFulfilled and IntentFulfilledWithCall events
const IntentFulfilledEventABI = `[
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "internalType": "bytes32", "name": "intentId", "type": "bytes32"},
			{"indexed": true, "internalType": "address", "name": "asset", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
			{"indexed": true, "internalType": "address", "name": "receiver", "type": "address"}
		],
		"name": "IntentFulfilled",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "internalType": "bytes32", "name": "intentId", "type": "bytes32"},
			{"indexed": true, "internalType": "address", "name": "asset", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
			{"indexed": true, "internalType": "address", "name": "receiver", "type": "address"},
			{"indexed": false, "internalType": "bytes", "name": "data", "type": "bytes"}
		],
		"name": "IntentFulfilledWithCall",
		"type": "event"
	}
]`

// IntentSettledEventABI is the ABI for the IntentSettled and IntentSettledWithCall events
const IntentSettledEventABI = `[
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "internalType": "bytes32", "name": "intentId", "type": "bytes32"},
			{"indexed": true, "internalType": "address", "name": "asset", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
			{"indexed": true, "internalType": "address", "name": "receiver", "type": "address"},
			{"indexed": false, "internalType": "bool", "name": "fulfilled", "type": "bool"},
			{"indexed": false, "internalType": "address", "name": "fulfiller", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "actualAmount", "type": "uint256"},
			{"indexed": false, "internalType": "uint256", "name": "paidTip", "type": "uint256"}
		],
		"name": "IntentSettled",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "internalType": "bytes32", "name": "intentId", "type": "bytes32"},
			{"indexed": true, "internalType": "address", "name": "asset", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "amount", "type": "uint256"},
			{"indexed": true, "internalType": "address", "name": "receiver", "type": "address"},
			{"indexed": false, "internalType": "bool", "name": "fulfilled", "type": "bool"},
			{"indexed": false, "internalType": "address", "name": "fulfiller", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "actualAmount", "type": "uint256"},
			{"indexed": false, "internalType": "uint256", "name": "paidTip", "type": "uint256"},
			{"indexed": false, "internalType": "bytes", "name": "data", "type": "bytes"}
		],
		"name": "IntentSettledWithCall",
		"type": "event"
	}
]`
