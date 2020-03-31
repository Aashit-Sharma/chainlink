package client

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// NewRoundEvent represents a NewRound event log that has
// been unmarshaled
// type NewRoundEvent struct {
// 	RoundID   *big.Int
// 	StartedBy common.Address
// }

// SubmissionReceivedEvent TODO...
// event SubmissionReceived(
//   int256 indexed answer,
//   uint32 indexed round,
//   address indexed oracle
// );
type SubmissionReceivedEvent struct {
	Answer  string
	RoundID *big.Int
	Oracle  common.Address
}
