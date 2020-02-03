package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetUtilityVersion(t *testing.T) {
	u := &utilityVersions{
		Prometheus: "",
		Nginx:      "",
		Fluentbit:  "",
	}

	setUtilityVersion(u, PROMETHEUS, "1.9")
	require.Equal(t, u.Prometheus, "1.9")

	setUtilityVersion(u, NGINX, "0.9")
	require.Equal(t, u.Prometheus, "1.9")
	require.Equal(t, u.Nginx, "0.9")

	setUtilityVersion(u, "an_error", "9")
	require.Equal(t, u.Prometheus, "1.9")
	require.Equal(t, u.Nginx, "0.9")
}

func TestGetUtilityVersion(t *testing.T) {
	u := &utilityVersions{
		Prometheus: "4",
		Nginx:      "5",
		Fluentbit:  "6",
	}

	require.Equal(t, getUtilityVersion(u, PROMETHEUS), "4")
	require.Equal(t, getUtilityVersion(u, NGINX), "5")
	require.Equal(t, getUtilityVersion(u, FLUENTBIT), "6")
	require.Equal(t, getUtilityVersion(u, "anything else"), "")
}

func TestSetActualVersion(t *testing.T) {
	c := &Cluster{
		Provider:            "aws",
		Provisioner:         "kops",
		ProviderMetadata:    []byte(`{"provider": "test1"}`),
		ProvisionerMetadata: []byte(`{"provisioner": "test1"}`),
		AllowInstallations:  false,
	}

	require.Equal(t, c.UtilityMetadata, []byte(nil))
	err := c.SetUtilityActualVersion(NGINX, "1.9.9")
	require.NoError(t, err)

	require.NotEqual(t, []byte(nil), c.UtilityMetadata)

	version, err := c.ActualUtilityVersion(NGINX)
	require.NoError(t, err)
	require.Equal(t, "1.9.9", version)
}

func TestSetDesired(t *testing.T) {
	c := &Cluster{
		Provider:            "aws",
		Provisioner:         "kops",
		ProviderMetadata:    []byte(`{"provider": "test1"}`),
		ProvisionerMetadata: []byte(`{"provisioner": "test1"}`),
		AllowInstallations:  false,
	}

	require.Equal(t, c.UtilityMetadata, []byte(nil))
	err := c.SetUtilityDesiredVersions(map[string]string{
		NGINX: "1.9.9",
	})
	require.NoError(t, err)

	require.NotEqual(t, []byte(nil), c.UtilityMetadata)

	version, err := c.DesiredUtilityVersion(NGINX)
	require.NoError(t, err)
	require.Equal(t, "1.9.9", version)

	version, err = c.DesiredUtilityVersion(PROMETHEUS)
	require.NoError(t, err)
	require.Equal(t, "", version)

}

func TestGetActualVersion(t *testing.T) {
	um := &UtilityMetadata{
		DesiredVersions: utilityVersions{
			Prometheus: "",
			Nginx:      "10.3",
			Fluentbit:  "1337",
		},
		ActualVersions: utilityVersions{
			Prometheus: "prometheus-10.3",
			Nginx:      "nginx-10.2",
			Fluentbit:  "fluent-bit-0.9",
		},
	}

	b, err := json.Marshal(um)
	require.NoError(t, err)
	require.NotEqual(t, 0, len(b))

	c := &Cluster{
		Provider:            "aws",
		Provisioner:         "kops",
		ProviderMetadata:    []byte(`{"provider": "test1"}`),
		ProvisionerMetadata: []byte(`{"provisioner": "test1"}`),
		AllowInstallations:  false,
		UtilityMetadata:     b,
	}

	require.NotEqual(t, 0, len(c.UtilityMetadata))

	version, err := c.ActualUtilityVersion(PROMETHEUS)
	require.NoError(t, err)
	require.Equal(t, "prometheus-10.3", version)

	version, err = c.ActualUtilityVersion(NGINX)
	require.NoError(t, err)
	require.Equal(t, "nginx-10.2", version)

	version, err = c.ActualUtilityVersion(FLUENTBIT)
	require.NoError(t, err)
	require.Equal(t, "fluent-bit-0.9", version)

	version, err = c.ActualUtilityVersion("something else that doesn't exist")
	require.NoError(t, err)
	require.Equal(t, "", version)
}

func TestGetDesiredVersion(t *testing.T) {
	um := &UtilityMetadata{
		DesiredVersions: utilityVersions{
			Prometheus: "",
			Nginx:      "10.3",
			Fluentbit:  "1337",
		},
		ActualVersions: utilityVersions{
			Prometheus: "prometheus-10.3",
			Nginx:      "nginx-10.2",
			Fluentbit:  "fluent-bit-0.9",
		},
	}

	b, err := json.Marshal(um)
	require.NoError(t, err)
	require.NotEqual(t, 0, len(b))

	c := &Cluster{
		Provider:            "aws",
		Provisioner:         "kops",
		ProviderMetadata:    []byte(`{"provider": "test1"}`),
		ProvisionerMetadata: []byte(`{"provisioner": "test1"}`),
		AllowInstallations:  false,
		UtilityMetadata:     b,
	}

	require.NotEqual(t, 0, len(c.UtilityMetadata))

	version, err := c.DesiredUtilityVersion(PROMETHEUS)
	require.NoError(t, err)
	require.Equal(t, "", version)

	version, err = c.DesiredUtilityVersion(NGINX)
	require.NoError(t, err)
	require.Equal(t, "10.3", version)

	version, err = c.DesiredUtilityVersion(FLUENTBIT)
	require.NoError(t, err)
	require.Equal(t, "1337", version)

	version, err = c.DesiredUtilityVersion("something else that doesn't exist")
	require.NoError(t, err)
	require.Equal(t, "", version)
}
