In order to instantiate Ethereum state and Ethereum Virtual Machine inside the Ethermint, we will do the following:

# Instantiate `core.BlockChain`
Here is the constructor signature:
```go
func NewBlockChain(
    db ethdb.Database,
    cacheConfig *CacheConfig,
    chainConfig *params.ChainConfig,
    engine consensus.Engine,
    vmConfig vm.Config,
) (*BlockChain, error) {
```
both `ethdb.Database` and `consensus.Engine` are interfaces. First one can be suitably implemented using `KVStore`, which really touching go-etherum code.
The second one is likely not required for what we want to do, i.e., create the `StateProcessor`.

# Instantiate `core.StateProcessor`
Here is the constructor signature:
```go
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}
```

# Apply transaction to the state
In order to apply transaction to the state, using an instance of `core.StateProcessor`, one needs to call this method:
```go
func ApplyTransaction(
	config *params.ChainConfig,
	bc *BlockChain,
	author *common.Address,
	gp *GasPool,
	statedb *state.StateDB,
	header *types.Header,
	tx *types.Transaction,
	usedGas *uint64,
	cfg vm.Config
) (*types.Receipt, uint64, error) {
```
It references concrete type `Header`:
```go
type Header struct {
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    common.Address `json:"miner"            gencodec:"required"`
	Root        common.Hash    `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"`
	Bloom       Bloom          `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int       `json:"difficulty"       gencodec:"required"`
	Number      *big.Int       `json:"number"           gencodec:"required"`
	GasLimit    uint64         `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"`
	Time        *big.Int       `json:"timestamp"        gencodec:"required"`
	Extra       []byte         `json:"extraData"        gencodec:"required"`
	MixDigest   common.Hash    `json:"mixHash"          gencodec:"required"`
	Nonce       BlockNonce     `json:"nonce"            gencodec:"required"`
}
```
Here, `common.Hash` is a type based on 32-byte array, and `common.Address` is based on 20-byte array. `BlockNonce` is based on 8-byte array.
`Bloom` is based on 256-byte array. As long as we produce hashes, addresses, nonces, and bloom filters of the same size, we should be able
to use this code without any modifications too.
Type `GasPool` is based on `uint64`.
Type `Transaction` is also concrete:
```go
type Transaction struct {
	data txdata
	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type txdata struct {
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
}
```

`state.StateDB` is also a concrete type, with the following constructor signature:
```go
func New(root common.Hash, db Database) (*StateDB, error) {
```
Type `Database` is an interface:
```go
// Database wraps access to tries and contract code.
type Database interface {
	// OpenTrie opens the main account trie.
	OpenTrie(root common.Hash) (Trie, error)

	// OpenStorageTrie opens the storage trie of an account.
	OpenStorageTrie(addrHash, root common.Hash) (Trie, error)

	// CopyTrie returns an independent copy of the given trie.
	CopyTrie(Trie) Trie

	// ContractCode retrieves a particular contract's code.
	ContractCode(addrHash, codeHash common.Hash) ([]byte, error)

	// ContractCodeSize retrieves a particular contracts code's size.
	ContractCodeSize(addrHash, codeHash common.Hash) (int, error)

	// TrieDB retrieves the low level trie database used for data storage.
	TrieDB() *trie.Database
}

// Trie is a Ethereum Merkle Trie.
type Trie interface {
	TryGet(key []byte) ([]byte, error)
	TryUpdate(key, value []byte) error
	TryDelete(key []byte) error
	Commit(onleaf trie.LeafCallback) (common.Hash, error)
	Hash() common.Hash
	NodeIterator(startKey []byte) trie.NodeIterator
	GetKey([]byte) []byte // TODO(fjl): remove this when SecureTrie is removed
	Prove(key []byte, fromLevel uint, proofDb ethdb.Putter) error
}
```
`trie.Database` is not an interface, but it can be instantiated using this constructor:
```go
func NewDatabase(diskdb ethdb.Database) *Database {
```
Both `trie.NodeIterator` and `ethdb.Putter` are interfaces that can be suitably implemented

# Conclusion
It should be possible to instantiate blockchain and state processor objects and plug them into KVStore and Tendermint consensus respectively.
This will be the next objective
 