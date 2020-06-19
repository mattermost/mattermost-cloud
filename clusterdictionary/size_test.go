package clusterdictionary

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestCheckSize(t *testing.T) {
	var testCases = []struct {
		size            string
		expectSupported bool
	}{
		{"", false},
		{"unknown", false},
		{SizeAlef500, true},
		{SizeAlef1000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.size, func(t *testing.T) {
			assert.Equal(t, tc.expectSupported, IsValidClusterSize(tc.size))
		})
	}
}

func TestApplyToCreateClusterRequest(t *testing.T) {
	var sizeTests = []struct {
		size        string
		request     *model.CreateClusterRequest
		expectError bool
	}{
		{
			"",
			&model.CreateClusterRequest{},
			false,
		}, {
			"InvalidSize",
			&model.CreateClusterRequest{},
			true,
		}, {
			SizeAlefDev,
			&model.CreateClusterRequest{
				MasterInstanceType: "t3.medium",
				MasterCount:        1,
				NodeInstanceType:   "t3.medium",
				NodeMinCount:       2,
				NodeMaxCount:       2,
			},
			false,
		}, {
			SizeAlef500,
			&model.CreateClusterRequest{
				MasterInstanceType: "t3.medium",
				MasterCount:        1,
				NodeInstanceType:   "m5.large",
				NodeMinCount:       2,
				NodeMaxCount:       2,
			},
			false,
		}, {
			SizeAlef1000,
			&model.CreateClusterRequest{
				MasterInstanceType: "t3.large",
				MasterCount:        1,
				NodeInstanceType:   "m5.large",
				NodeMinCount:       4,
				NodeMaxCount:       4,
			},
			false,
		}, {
			SizeAlef5000,
			&model.CreateClusterRequest{
				MasterInstanceType: "t3.large",
				MasterCount:        1,
				NodeInstanceType:   "m5.large",
				NodeMinCount:       6,
				NodeMaxCount:       6,
			},
			false,
		}, {
			SizeAlef10000,
			&model.CreateClusterRequest{
				MasterInstanceType: "t3.large",
				MasterCount:        3,
				NodeInstanceType:   "m5.large",
				NodeMinCount:       10,
				NodeMaxCount:       10,
			},
			false,
		},
	}

	for _, tt := range sizeTests {
		t.Run(tt.size, func(t *testing.T) {
			request := &model.CreateClusterRequest{}
			err := ApplyToCreateClusterRequest(tt.size, request)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.request, request)
		})
	}
}

func TestApplyToPatchClusterSizeRequest(t *testing.T) {
	var sizeTests = []struct {
		size        string
		request     *model.PatchClusterSizeRequest
		expectError bool
	}{
		{
			"",
			&model.PatchClusterSizeRequest{},
			false,
		}, {
			"InvalidSize",
			&model.PatchClusterSizeRequest{},
			true,
		}, {
			SizeAlefDev,
			&model.PatchClusterSizeRequest{
				NodeInstanceType: stringToPointer("t3.medium"),
				NodeMinCount:     int64ToPointer(2),
				NodeMaxCount:     int64ToPointer(2),
			},
			false,
		}, {
			SizeAlef500,
			&model.PatchClusterSizeRequest{
				NodeInstanceType: stringToPointer("m5.large"),
				NodeMinCount:     int64ToPointer(2),
				NodeMaxCount:     int64ToPointer(2),
			},
			false,
		}, {
			SizeAlef1000,
			&model.PatchClusterSizeRequest{
				NodeInstanceType: stringToPointer("m5.large"),
				NodeMinCount:     int64ToPointer(4),
				NodeMaxCount:     int64ToPointer(4),
			},
			false,
		}, {
			SizeAlef5000,
			&model.PatchClusterSizeRequest{
				NodeInstanceType: stringToPointer("m5.large"),
				NodeMinCount:     int64ToPointer(6),
				NodeMaxCount:     int64ToPointer(6),
			},
			false,
		}, {
			SizeAlef10000,
			&model.PatchClusterSizeRequest{
				NodeInstanceType: stringToPointer("m5.large"),
				NodeMinCount:     int64ToPointer(10),
				NodeMaxCount:     int64ToPointer(10),
			},
			false,
		},
	}

	for _, tt := range sizeTests {
		t.Run(tt.size, func(t *testing.T) {
			request := &model.PatchClusterSizeRequest{}
			err := ApplyToPatchClusterSizeRequest(tt.size, request)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.request, request)
		})
	}
}

func stringToPointer(s string) *string {
	return &s
}

func int64ToPointer(i int64) *int64 {
	return &i
}
