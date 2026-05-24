package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ViperSuite struct {
	suite.Suite
	originalEnv string
	envSet      bool
}

func (s *ViperSuite) SetupTest() {
	s.originalEnv, s.envSet = os.LookupEnv("APP_ENV")
}

func (s *ViperSuite) TearDownTest() {
	if s.envSet {
		s.Require().NoError(os.Setenv("APP_ENV", s.originalEnv))
	} else {
		s.Require().NoError(os.Unsetenv("APP_ENV"))
	}
}

func (s *ViperSuite) setEnv(value string) {
	s.Require().NoError(os.Setenv("APP_ENV", value))
}

func (s *ViperSuite) TestDetermineEnvironment_EmptyDefaultsToDev() {
	s.setEnv("")
	assert.Equal(s.T(), "dev", DetermineEnvironment())
}

func (s *ViperSuite) TestDetermineEnvironment_ExplicitDev() {
	s.setEnv("dev")
	assert.Equal(s.T(), "dev", DetermineEnvironment())
}

func (s *ViperSuite) TestDetermineEnvironment_Prod() {
	s.setEnv("prod")
	assert.Equal(s.T(), "prod", DetermineEnvironment())
}

func (s *ViperSuite) TestDetermineEnvironment_Test() {
	s.setEnv("test")
	assert.Equal(s.T(), "test", DetermineEnvironment())
}

func (s *ViperSuite) TestDetermineEnvironment_UnknownDefaultsToDev() {
	s.setEnv("staging")
	assert.Equal(s.T(), "dev", DetermineEnvironment())
}

func TestViperSuite(t *testing.T) {
	suite.Run(t, new(ViperSuite))
}
