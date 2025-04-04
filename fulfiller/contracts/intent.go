package contracts

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// IntentABI is the ABI of the Intent contract
const IntentABI = `[
	{
		"inputs": [
			{
				"internalType": "bytes32",
				"name": "intentId",
				"type": "bytes32"
			},
			{
				"internalType": "address",
				"name": "asset",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "amount",
				"type": "uint256"
			},
			{
				"internalType": "address",
				"name": "receiver",
				"type": "address"
			}
		],
		"name": "fulfill",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
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
				"indexed": true,
				"internalType": "address",
				"name": "receiver",
				"type": "address"
			}
		],
		"name": "IntentFulfilled",
		"type": "event"
	}
]`

// Intent is an auto generated Go binding around an Ethereum contract.
type Intent struct {
	IntentCaller     // Read-only binding to the contract
	IntentTransactor // Write-only binding to the contract
	IntentFilterer   // Log filterer for contract events
}

// IntentCaller is an auto generated read-only Go binding around an Ethereum contract.
type IntentCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IntentTransactor is an auto generated write-only Go binding around an Ethereum contract.
type IntentTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IntentFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type IntentFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// IntentSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type IntentSession struct {
	Contract     *Intent           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IntentCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type IntentCallerSession struct {
	Contract *IntentCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// IntentTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type IntentTransactorSession struct {
	Contract     *IntentTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// IntentRaw is an auto generated low-level Go binding around an Ethereum contract.
type IntentRaw struct {
	Contract *Intent // Generic contract binding to access the raw methods on
}

// IntentCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type IntentCallerRaw struct {
	Contract *IntentCaller // Generic read-only contract binding to access the raw methods on
}

// IntentTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type IntentTransactorRaw struct {
	Contract *IntentTransactor // Generic write-only contract binding to access the raw methods on
}

// NewIntent creates a new instance of Intent, bound to a specific deployed contract.
func NewIntent(address common.Address, backend bind.ContractBackend) (*Intent, error) {
	contract, err := bindIntent(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Intent{IntentCaller: IntentCaller{contract: contract}, IntentTransactor: IntentTransactor{contract: contract}, IntentFilterer: IntentFilterer{contract: contract}}, nil
}

// NewIntentCaller creates a new read-only instance of Intent, bound to a specific deployed contract.
func NewIntentCaller(address common.Address, caller bind.ContractCaller) (*IntentCaller, error) {
	contract, err := bindIntent(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &IntentCaller{contract: contract}, nil
}

// NewIntentTransactor creates a new write-only instance of Intent, bound to a specific deployed contract.
func NewIntentTransactor(address common.Address, transactor bind.ContractTransactor) (*IntentTransactor, error) {
	contract, err := bindIntent(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &IntentTransactor{contract: contract}, nil
}

// NewIntentFilterer creates a new log filterer instance of Intent, bound to a specific deployed contract.
func NewIntentFilterer(address common.Address, filterer bind.ContractFilterer) (*IntentFilterer, error) {
	contract, err := bindIntent(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &IntentFilterer{contract: contract}, nil
}

// bindIntent binds a generic wrapper to an already deployed contract.
func bindIntent(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(IntentABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Intent *IntentRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Intent.Contract.IntentCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Intent *IntentRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Intent.Contract.IntentTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Intent *IntentRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Intent.Contract.IntentTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Intent *IntentCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Intent.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Intent *IntentTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Intent.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Intent *IntentTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Intent.Contract.contract.Transact(opts, method, params...)
}

// Fulfill is a paid mutator transaction binding the contract method 0x12345678.
//
// Solidity: function fulfill(bytes32 intentId, address asset, uint256 amount, address receiver) returns()
func (_Intent *IntentTransactor) Fulfill(opts *bind.TransactOpts, intentId [32]byte, asset common.Address, amount *big.Int, receiver common.Address) (*types.Transaction, error) {
	return _Intent.contract.Transact(opts, "fulfill", intentId, asset, amount, receiver)
}

// Fulfill is a paid mutator transaction binding the contract method 0x12345678.
//
// Solidity: function fulfill(bytes32 intentId, address asset, uint256 amount, address receiver) returns()
func (_Intent *IntentSession) Fulfill(intentId [32]byte, asset common.Address, amount *big.Int, receiver common.Address) (*types.Transaction, error) {
	return _Intent.Contract.Fulfill(&_Intent.TransactOpts, intentId, asset, amount, receiver)
}

// Fulfill is a paid mutator transaction binding the contract method 0x12345678.
//
// Solidity: function fulfill(bytes32 intentId, address asset, uint256 amount, address receiver) returns()
func (_Intent *IntentTransactorSession) Fulfill(intentId [32]byte, asset common.Address, amount *big.Int, receiver common.Address) (*types.Transaction, error) {
	return _Intent.Contract.Fulfill(&_Intent.TransactOpts, intentId, asset, amount, receiver)
}

// IntentIntentFulfilledIterator is returned from FilterIntentFulfilled and is used to iterate over the raw logs and unpacked data for IntentFulfilled events raised by the Intent contract.
type IntentIntentFulfilledIterator struct {
	Event *IntentIntentFulfilled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *IntentIntentFulfilledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(IntentIntentFulfilled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(IntentIntentFulfilled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *IntentIntentFulfilledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *IntentIntentFulfilledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// IntentIntentFulfilled represents a IntentFulfilled event raised by the Intent contract.
type IntentIntentFulfilled struct {
	IntentId [32]byte
	Asset    common.Address
	Amount   *big.Int
	Receiver common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterIntentFulfilled is a free log retrieval operation binding the contract event 0x12345678.
//
// Solidity: event IntentFulfilled(bytes32 indexed intentId, address indexed asset, uint256 amount, address indexed receiver)
func (_Intent *IntentFilterer) FilterIntentFulfilled(opts *bind.FilterOpts, intentId [][32]byte, asset []common.Address, receiver []common.Address) (*IntentIntentFulfilledIterator, error) {
	var intentIdRule []interface{}
	for _, intentIdItem := range intentId {
		intentIdRule = append(intentIdRule, intentIdItem)
	}
	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}
	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _Intent.contract.FilterLogs(opts, "IntentFulfilled", intentIdRule, assetRule, receiverRule)
	if err != nil {
		return nil, err
	}
	return &IntentIntentFulfilledIterator{contract: _Intent.contract, event: "IntentFulfilled", logs: logs, sub: sub}, nil
}

// WatchIntentFulfilled is a free log subscription operation binding the contract event 0x12345678.
//
// Solidity: event IntentFulfilled(bytes32 indexed intentId, address indexed asset, uint256 amount, address indexed receiver)
func (_Intent *IntentFilterer) WatchIntentFulfilled(opts *bind.WatchOpts, sink chan<- *IntentIntentFulfilled, intentId [][32]byte, asset []common.Address, receiver []common.Address) (event.Subscription, error) {
	var intentIdRule []interface{}
	for _, intentIdItem := range intentId {
		intentIdRule = append(intentIdRule, intentIdItem)
	}
	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}
	var receiverRule []interface{}
	for _, receiverItem := range receiver {
		receiverRule = append(receiverRule, receiverItem)
	}

	logs, sub, err := _Intent.contract.WatchLogs(opts, "IntentFulfilled", intentIdRule, assetRule, receiverRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(IntentIntentFulfilled)
				if err := _Intent.contract.UnpackLog(event, "IntentFulfilled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseIntentFulfilled is a log parse operation binding the contract event 0x12345678.
//
// Solidity: event IntentFulfilled(bytes32 indexed intentId, address indexed asset, uint256 amount, address indexed receiver)
func (_Intent *IntentFilterer) ParseIntentFulfilled(log types.Log) (*IntentIntentFulfilled, error) {
	event := new(IntentIntentFulfilled)
	if err := _Intent.contract.UnpackLog(event, "IntentFulfilled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
