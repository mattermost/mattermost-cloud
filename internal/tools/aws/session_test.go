package aws

func (a *AWSTestSuite) TestSingletonSession() {
	sess, err := NewAWSSession()
	a.Assert().NotNil(sess)
	a.Assert().NoError(err)

	sess2, _ := NewAWSSession()
	a.Assert().NotNil(sess2)
	a.Assert().NoError(err)

	a.Assert().Equal(sess, sess2)
}
