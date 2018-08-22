package actor_test

import (
	"context"
	"math/big"
	"testing"

	"gx/ipfs/QmQZadYTDF4ud9DdK85PH2vReJRzUM9YfVW4ReB1q2m51p/go-hamt-ipld"
	"gx/ipfs/QmVG5gxteQNEMhrS8prJSmU2C9rebtFuTd3SYZ5kE3YZ5k/go-datastore"
	"gx/ipfs/QmcmpX42gtDv1fz24kau4wjS9hfwWj5VexWBKgGnWzsyag/go-ipfs-blockstore"

	. "github.com/filecoin-project/go-filecoin/actor"
	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/filecoin-project/go-filecoin/vm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActorMarshal(t *testing.T) {
	assert := assert.New(t)
	actor := NewActor(types.AccountActorCodeCid, types.NewAttoFILFromFIL(1))
	actor.Head = requireCid(t, "Actor Storage")
	actor.IncNonce()

	marshalled, err := actor.Marshal()
	assert.NoError(err)

	actorBack := Actor{}
	err = actorBack.Unmarshal(marshalled)
	assert.NoError(err)

	assert.Equal(actor.Code, actorBack.Code)
	assert.Equal(actor.Head, actorBack.Head)
	assert.Equal(actor.Nonce, actorBack.Nonce)

	c1, err := actor.Cid()
	assert.NoError(err)
	c2, err := actorBack.Cid()
	assert.NoError(err)
	assert.Equal(c1, c2)
}

func TestMarshalValue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		assert := assert.New(t)

		testCases := []struct {
			In  interface{}
			Out []byte
		}{
			{In: []byte("hello"), Out: []byte("hello")},
			{In: big.NewInt(100), Out: big.NewInt(100).Bytes()},
			{In: "hello", Out: []byte("hello")},
		}

		for _, tc := range testCases {
			out, err := MarshalValue(tc.In)
			assert.NoError(err)
			assert.Equal(out, tc.Out)
		}
	})

	t.Run("failure", func(t *testing.T) {
		assert := assert.New(t)

		out, err := MarshalValue(big.NewRat(1, 2))
		assert.Equal(err.Error(), "unknown type: *big.Rat")
		assert.Nil(out)
	})
}

func TestLoadLookup(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	ds := datastore.NewMapDatastore()
	bs := blockstore.NewBlockstore(ds)
	vms := vm.NewStorageMap(bs)
	storage := vms.NewStorage(address.TestAddress, &Actor{})
	ctx := context.TODO()

	lookup, err := LoadLookup(ctx, storage, nil)
	require.NoError(err)

	err = lookup.Set(ctx, "foo", "someData")
	require.NoError(err)

	cid, err := lookup.Commit(ctx)
	require.NoError(err)

	assert.NotNil(cid)

	err = storage.Commit(cid, nil)
	require.NoError(err)

	err = vms.Flush()
	require.NoError(err)

	t.Run("Fetch chunk by cid", func(t *testing.T) {
		bs = blockstore.NewBlockstore(ds)
		vms = vm.NewStorageMap(bs)
		storage = vms.NewStorage(address.TestAddress, &Actor{})

		lookup, err = LoadLookup(ctx, storage, cid)
		require.NoError(err)

		value, err := lookup.Find(ctx, "foo")
		require.NoError(err)

		assert.Equal("someData", value)
	})

	t.Run("Get errs for missing key", func(t *testing.T) {
		bs = blockstore.NewBlockstore(ds)
		vms = vm.NewStorageMap(bs)
		storage = vms.NewStorage(address.TestAddress, &Actor{})

		lookup, err = LoadLookup(ctx, storage, cid)
		require.NoError(err)

		_, err := lookup.Find(ctx, "bar")
		require.Error(err)
		assert.Equal(hamt.ErrNotFound, err)
	})
}

func TestLoadLookupWithInvalidCid(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	ds := datastore.NewMapDatastore()
	bs := blockstore.NewBlockstore(ds)
	vms := vm.NewStorageMap(bs)
	storage := vms.NewStorage(address.TestAddress, &Actor{})
	ctx := context.TODO()

	cid := types.NewCidForTestGetter()()

	_, err := LoadLookup(ctx, storage, cid)
	require.Error(err)
	assert.Equal(vm.ErrNotFound, err)
}

func TestSetKeyValue(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	ds := datastore.NewMapDatastore()
	bs := blockstore.NewBlockstore(ds)
	vms := vm.NewStorageMap(bs)
	storage := vms.NewStorage(address.TestAddress, &Actor{})
	ctx := context.TODO()

	cid, err := SetKeyValue(ctx, storage, nil, "foo", "bar")
	require.NoError(err)
	assert.NotNil(cid)

	lookup, err := LoadLookup(ctx, storage, cid)
	require.NoError(err)

	val, err := lookup.Find(ctx, "foo")
	require.NoError(err)
	assert.Equal("bar", val)
}