# Terminology

This page explains key terms used throughout the Speedrun documentation.

## Core Concepts

<div class="terminology-list">
  <div class="term-container">
    <div class="term">Intent</div>
    <div class="definition">
      An intent represents the desire to complete an action on a target chain, currently focused on token transfers. It captures what a user wants to accomplish without specifying exactly how it should be done.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Initiate</div>
    <div class="definition">
      The act of starting an intent from the user perspective. The person who creates the intent is called the "initiator" and they specify what they want to accomplish.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Fulfill</div>
    <div class="definition">
      The process where a "fulfiller" directly completes the intent on the target chain. Fulfillers help execute the requested action, often providing liquidity to make the transaction happen.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Settle</div>
    <div class="definition">
      The settlement of the intent, when the funds are finally received on the target chain. This marks the successful completion of the entire intent process.
    </div>
  </div>
</div>

## Technical Terms

<div class="terminology-list">
  <div class="term-container">
    <div class="term">Source Chain</div>
    <div class="definition">
      The blockchain where the intent is initiated and the original tokens are held. This is where users start their cross-chain transfer.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Target Chain</div>
    <div class="definition">
      The destination blockchain where the tokens will be received. This is where fulfillers execute the transfer.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Intent Fee</div>
    <div class="definition">
      A fee paid to fulfillers for executing the intent. This fee is set by the initiator and can be used to incentivize faster fulfillment.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Fulfiller</div>
    <div class="definition">
      A third-party agent who executes intents on the target chain. Fulfillers provide liquidity and receive intent fees for their service.
    </div>
  </div>
</div>

## Status Terms

<div class="terminology-list">
  <div class="term-container">
    <div class="term">Pending</div>
    <div class="definition">
      The initial state of an intent after it has been created but before it is fulfilled.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Fulfilled</div>
    <div class="definition">
      The state when a fulfiller has executed the intent on the target chain but before final settlement.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Settled</div>
    <div class="definition">
      The final state when the intent has been completely processed and settled on the target chain.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Failed</div>
    <div class="definition">
      The state when an intent could not be fulfilled or settled successfully.
    </div>
  </div>
</div>

## API Terms

<div class="terminology-list">
  <div class="term-container">
    <div class="term">Intent ID</div>
    <div class="definition">
      A unique identifier for each intent, used to track and reference intents across the system.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Transaction Hash</div>
    <div class="definition">
      The unique identifier of a blockchain transaction, used to track the fulfillment and settlement of intents.
    </div>
  </div>

  <div class="term-container">
    <div class="term">Block Number</div>
    <div class="definition">
      The sequential number of a block in the blockchain, used for tracking the progress of intent processing.
    </div>
  </div>
</div>
