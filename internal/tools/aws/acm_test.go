package aws

var (
	testARNCertificate          = "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"
	testParsedCertificateTagKey = "MattermostCloudInstallationCertificates"
	testCertificateTagValue     = "true"
)

// func (api *mockAPI) listCertificates(*acm.ACM, *acm.ListCertificatesInput) (*acm.ListCertificatesOutput, error) {
// 	return &acm.ListCertificatesOutput{
// 		CertificateSummaryList: []*acm.CertificateSummary{&acm.CertificateSummary{
// 			CertificateArn: &testARNCertificate,
// 			DomainName:     &testDNSName,
// 		}},
// 	}, api.returnedError
// }

// func (api *mockAPI) listTagsForCertificate(*acm.ACM, *acm.ListTagsForCertificateInput) (*acm.ListTagsForCertificateOutput, error) {
// 	return &acm.ListTagsForCertificateOutput{
// 		Tags: []*acm.Tag{&acm.Tag{Key: &testParsedCertificateTagKey, Value: &testCertificateTagValue}},
// 	}, api.returnedError
// }

// func TestGetCertificateByTag(t *testing.T) {
// 	a := Client{api: &mockAPI{}}
// 	list, err := a.GetCertificateSummaryByTag(testParsedCertificateTagKey, testCertificateTagValue)
// 	assert.NoError(t, err)
// 	assert.Equal(t, *list.CertificateArn, testARNCertificate)
// }

// func TestGetCertificateByTagError(t *testing.T) {
// 	a := Client{api: &mockAPI{returnedError: errors.New("something went wrong")}}
// 	_, err := a.GetCertificateSummaryByTag(testParsedRoute53TagKey, testRoute53TagValue)
// 	assert.Error(t, err)
// }

// func TestGetCertificateByTagWrongKey(t *testing.T) {
// 	a := Client{api: &mockAPI{}}
// 	_, err := a.GetCertificateSummaryByTag("banana", testRoute53TagValue)
// 	assert.Error(t, err)
// }

// func TestGetCertificateByTagEmptyValue(t *testing.T) {
// 	a := Client{api: &mockAPI{}}
// 	_, err := a.GetCertificateSummaryByTag(testParsedRoute53TagKey, "")
// 	assert.Error(t, err)
// }
