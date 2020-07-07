package keeper

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

type Querier struct {
	Keeper
}

var _ types.QueryServer = Querier{}

func (k Querier) Validators(c context.Context, req *types.QueryValidatorsRequest) (*types.QueryValidatorsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.Status == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	var validators types.Validators
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	valStore := prefix.NewStore(store, types.ValidatorsKey)

	res, err := query.FilteredPaginate(valStore, req.Req, func(key []byte, value []byte, accumulate bool) (bool, error) {
		val, err := types.UnmarshalValidator(k.cdc, value)
		if err != nil {
			return false, err
		}

		if strings.EqualFold(val.GetStatus().String(), req.Status) {
			if accumulate {
				validators = append(validators, val)
			}

			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return &types.QueryValidatorsResponse{}, err
	}

	return &types.QueryValidatorsResponse{Validators: validators, Res: res}, nil
}

func (k Querier) Validator(c context.Context, req *types.QueryValidatorRequest) (*types.QueryValidatorResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	validator, found := k.GetValidator(ctx, req.ValidatorAddr)
	if !found {
		return nil, status.Errorf(codes.NotFound, "validator %s not found", req.ValidatorAddr)
	}

	return &types.QueryValidatorResponse{Validator: validator}, nil
}

func (k Querier) ValidatorDelegations(c context.Context, req *types.QueryValidatorDelegationsRequest) (*types.QueryValidatorDelegationsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.ValidatorAddr == nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	delegations := k.GetValidatorDelegations(ctx, req.ValidatorAddr)
	store := ctx.KVStore(k.storeKey)
	valStore := prefix.NewStore(store, types.DelegationKey)
	res, err := query.FilteredPaginate(valStore, req.Req, func(key []byte, value []byte, accumulate bool) (bool, error) {
		delegation, err := types.UnmarshalDelegation(k.cdc, value)
		if err != nil {
			return false, err
		}
		if delegation.GetValidatorAddr().Equals(req.ValidatorAddr) {
			if accumulate {
				delegations = append(delegations, delegation)
			}
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Unable to find delegations for"+
			" validator %d", req.ValidatorAddr)
	}
	delResponses, err := DelegationsToDelegationResponses(ctx, k.Keeper, delegations)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Unable to convert delegations")
	}

	return &types.QueryValidatorDelegationsResponse{DelegationResponses: delResponses, Res: res}, nil
}

func (k Querier) ValidatorUnbondingDelegations(c context.Context, req *types.QueryValidatorUnbondingDelegationsRequest) (*types.QueryValidatorUnbondingDelegationsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.ValidatorAddr == nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)
	ubdStore := prefix.NewStore(store, types.GetUBDsByValIndexKey(req.ValidatorAddr))
	var ubds types.UnbondingDelegations
	res, err := query.Paginate(ubdStore, req.Req, func(key []byte, value []byte) error {
		ubd, err := types.UnmarshalUBD(k.cdc, value)
		if err != nil {
			return err
		}
		ubds = append(ubds, ubd)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorUnbondingDelegationsResponse{UnbondingResponses: ubds, Res: res}, nil
}

func (k Querier) Delegation(c context.Context, req *types.QueryDelegationRequest) (*types.QueryDelegationResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == nil || req.ValidatorAddr == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	delegation, found := k.GetDelegation(ctx, req.DelegatorAddr, req.ValidatorAddr)
	if !found {
		return nil, status.Errorf(codes.NotFound, "delegation with delegator %s not found for"+
			"valiedator %s", req.DelegatorAddr, req.ValidatorAddr)
	}

	delResponse, err := DelegationToDelegationResponse(ctx, k.Keeper, delegation)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Unable to convert delegations")
	}

	return &types.QueryDelegationResponse{DelegationResponse: &delResponse}, nil
}

func (k Querier) UnbondingDelegation(c context.Context, req *types.QueryUnbondingDelegationRequest) (*types.QueryUnbondingDelegationResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	unbond, found := k.GetUnbondingDelegation(ctx, req.DelegatorAddr, req.ValidatorAddr)
	if !found {
		return nil, status.Errorf(codes.NotFound, "unbonding delegation with delegator %s not found for"+
			"valiedator %s", req.DelegatorAddr, req.ValidatorAddr)
	}

	return &types.QueryUnbondingDelegationResponse{Unbond: unbond}, nil
}

func (k Querier) DelegatorDelegations(c context.Context, req *types.QueryDelegatorDelegationsRequest) (*types.QueryDelegatorDelegationsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request")
	}

	var delegations types.Delegations
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	delStore := prefix.NewStore(store, types.GetDelegationsKey(req.DelegatorAddr))

	res, err := query.Paginate(delStore, req.Req, func(key []byte, value []byte) error {
		delegation, err := types.UnmarshalDelegation(k.cdc, value)
		if err != nil {
			return err
		}
		delegations = append(delegations, delegation)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if delegations == nil {
		return &types.QueryDelegatorDelegationsResponse{DelegationResponses: types.DelegationResponses{}}, nil
	}
	delegationResps, err := DelegationsToDelegationResponses(ctx, k.Keeper, delegations)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegatorDelegationsResponse{DelegationResponses: delegationResps, Res: res}, nil

}

func (k Querier) DelegatorValidator(c context.Context, req *types.QueryDelegatorValidatorRequest) (*types.QueryDelegatorValidatorResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == nil || req.ValidatorAddr == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	validator, err := k.GetDelegatorValidator(ctx, req.DelegatorAddr, req.ValidatorAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegatorValidatorResponse{Validator: validator}, nil
}

func (k Querier) DelegatorUnbondingDelegations(c context.Context, req *types.QueryDelegatorUnbondingDelegationsRequest) (*types.QueryDelegatorUnbondingDelegationsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request")
	}
	var unbondingDelegations types.UnbondingDelegations
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	unbStore := prefix.NewStore(store, types.GetUBDsKey(req.DelegatorAddr))

	res, err := query.Paginate(unbStore, req.Req, func(key []byte, value []byte) error {
		unbond, err := types.UnmarshalUBD(k.cdc, value)
		if err != nil {
			return err
		}
		unbondingDelegations = append(unbondingDelegations, unbond)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegatorUnbondingDelegationsResponse{UnbondingResponses: unbondingDelegations, Res: res}, nil
}

func (k Querier) HistoricalInfo(c context.Context, req *types.QueryHistoricalInfoRequest) (*types.QueryHistoricalInfoResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	hi, found := k.GetHistoricalInfo(ctx, req.Height)
	if !found {
		return nil, types.ErrNoHistoricalInfo
	}

	return &types.QueryHistoricalInfoResponse{Hist: &hi}, nil
}

func (k Querier) Redelegations(c context.Context, req *types.QueryRedelegationsRequest) (*types.QueryRedelegationsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	var redels types.Redelegations
	var res *query.PageResponse
	var err error

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)
	switch {
	case !req.DelegatorAddr.Empty() && !req.SrcValidatorAddr.Empty() && !req.DstValidatorAddr.Empty():
		redels, res, err = queryRedelegation(store, k, req)
		if err != nil {
			return nil, types.ErrNoRedelegation
		}
	case req.DelegatorAddr.Empty() && !req.SrcValidatorAddr.Empty() && req.DstValidatorAddr.Empty():
		redels, res, err = queryRedelegationsFromSrcValidator(store, k, req)
	default:
		redels, res, err = queryAllRedelegations(store, k, req)
	}
	if err != nil {
		return nil, err
	}
	redelResponses, err := RedelegationsToRedelegationResponses(ctx, k.Keeper, redels)
	if err != nil {
		return nil, err
	}

	return &types.QueryRedelegationsResponse{RedelegationResponses: redelResponses, Res: res}, nil
}

func (k Querier) DelegatorValidators(c context.Context, req *types.QueryDelegatorValidatorsRequest) (*types.QueryDelegatorValidatorsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr.Empty() {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	delStore := prefix.NewStore(store, types.GetDelegationsKey(req.DelegatorAddr))

	var validators types.Validators
	res, err := query.Paginate(delStore, req.Req, func(key []byte, value []byte) error {

		delegation, err := types.UnmarshalDelegation(k.cdc, value)
		if err != nil {
			return err
		}
		validator, found := k.GetValidator(ctx, delegation.ValidatorAddress)
		if !found {
			panic(types.ErrNoValidatorFound)
		}

		validators = append(validators, validator)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &types.QueryDelegatorValidatorsResponse{Validators: validators, Res: res}, nil
}

func (k Querier) Pool(c context.Context, _ *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	bondDenom := k.BondDenom(ctx)
	bondedPool := k.GetBondedPool(ctx)
	notBondedPool := k.GetNotBondedPool(ctx)

	if bondedPool == nil || notBondedPool == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "pool accounts haven't been set")
	}

	pool := types.NewPool(
		k.bankKeeper.GetBalance(ctx, notBondedPool.GetAddress(), bondDenom).Amount,
		k.bankKeeper.GetBalance(ctx, bondedPool.GetAddress(), bondDenom).Amount,
	)

	return &types.QueryPoolResponse{Pool: pool}, nil
}

func (k Querier) Parameters(c context.Context, _ *types.QueryParametersRequest) (*types.QueryParametersResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &types.QueryParametersResponse{Params: params}, nil
}

func queryRedelegation(store sdk.KVStore, k Querier, req *types.QueryRedelegationsRequest) (redels types.Redelegations, res *query.PageResponse, err error) {
	redStore := prefix.NewStore(store, types.GetREDKey(req.DelegatorAddr, req.SrcValidatorAddr, req.DstValidatorAddr))

	var redel types.Redelegation
	res, err = query.Paginate(redStore, req.Req, func(key []byte, value []byte) error {
		redel, err = types.UnmarshalRED(k.cdc, value)
		if err != nil {
			return err
		}
		return nil
	})

	redels = []types.Redelegation{redel}
	return redels, res, err
}

func queryRedelegationsFromSrcValidator(store sdk.KVStore, k Querier, req *types.QueryRedelegationsRequest) (redels types.Redelegations, res *query.PageResponse, err error) {
	srcValPrefix := types.GetREDsFromValSrcIndexKey(req.SrcValidatorAddr)
	redStore := prefix.NewStore(store, srcValPrefix)
	res, err = query.Paginate(redStore, req.Req, func(key []byte, value []byte) error {
		storeKey := types.GetREDKeyFromValSrcIndexKey(append(srcValPrefix, key...))
		storeValue := store.Get(storeKey)
		red, err := types.UnmarshalRED(k.cdc, storeValue)
		if err != nil {
			return err
		}
		redels = append(redels, red)
		return nil
	})

	return redels, res, err
}

func queryAllRedelegations(store sdk.KVStore, k Querier, req *types.QueryRedelegationsRequest) (redels types.Redelegations, res *query.PageResponse, err error) {
	redStore := prefix.NewStore(store, types.GetREDsKey(req.DelegatorAddr))
	res, err = query.Paginate(redStore, req.Req, func(key []byte, value []byte) error {
		redelegation, err := types.UnmarshalRED(k.cdc, value)
		if err != nil {
			return err
		}
		redels = append(redels, redelegation)
		return nil
	})

	return redels, res, err
}