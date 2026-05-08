// Package anp is a minimal Agent Network Protocol implementation.
//
// What's covered:
//
//   - Service / Registry types
//   - Register / Unregister / Discover / GetBest
//   - Default scoreFn = HealthScore / (1 + load)
//   - AsAgentTool: 5-action JSON tool wrapper
//
// What's NOT covered (out of scope for Phase 5 minimal):
//
//   - DID (decentralized identity) / private-key signing / public-key verification
//   - Trust evaluation / reputation
//   - Cross-network discovery / gossip / DHT
//   - Active health probing
//
// This is single-process, in-memory, learning scope only — NOT a real
// peer-to-peer network. Real ANP nodes would need at minimum the DID
// + signing layer to be safely federated.
//
// # Portability
//
// anp inherits the agents/pkg/llm portability contract.
package anp
