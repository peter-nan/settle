package functional

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/test"
	"github.com/stretchr/testify/assert"
)

func setupCreateAsset(
	t *testing.T,
) ([]*test.Mint, []*test.MintUser) {
	m := []*test.Mint{
		test.CreateMint(t),
	}

	u := []*test.MintUser{
		m[0].CreateUser(t),
	}

	return m, u
}

func TestCreateAsset(
	t *testing.T,
) {
	_, u := setupCreateAsset(t)

	status, raw := u[0].Post(t,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"2"},
		})

	var asset mint.AssetResource
	if err := raw.Extract("asset", &asset); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 201, status)
	assert.Regexp(t, mint.IDRegexp, asset.ID)
	assert.WithinDuration(t,
		time.Now(),
		time.Unix(0, asset.Created*1000*1000), 2*time.Millisecond)
	assert.Equal(t, u[0].Address, asset.Owner)

	assert.Regexp(t, mint.AssetNameRegexp, asset.Name)
	assert.Equal(t, fmt.Sprintf("%s[USD.2]", u[0].Address), asset.Name)
	assert.Equal(t, "USD", asset.Code)
	assert.Equal(t, int8(2), asset.Scale)
}

func TestCreateAssetWithInvalidCode(
	t *testing.T,
) {
	_, u := setupCreateAsset(t)

	status, raw := u[0].Post(t,
		"/assets",
		url.Values{
			"code":  {"U/S[D"},
			"scale": {"2"},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "code_invalid", e.ErrCode)
}

func TestCreateAssetWithInvalidScale(
	t *testing.T,
) {
	_, u := setupCreateAsset(t)

	status, raw := u[0].Post(t,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"221323132122"},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "scale_invalid", e.ErrCode)
}

func TestCreateAssetThatAlreadyExists(
	t *testing.T,
) {
	_, u := setupCreateAsset(t)

	status, _ := u[0].Post(t,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"2"},
		})
	assert.Equal(t, 201, status)

	status, raw := u[0].Post(t,
		"/assets",
		url.Values{
			"code":  {"USD"},
			"scale": {"2"},
		})

	var e errors.ConcreteUserError
	if err := raw.Extract("error", &e); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 400, status)
	assert.Equal(t, "asset_already_exists", e.ErrCode)
}
