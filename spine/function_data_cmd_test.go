package spine

import (
	"testing"

	"github.com/enbility/eebus-go/spine/model"
	"github.com/enbility/eebus-go/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestFunctionDataCmdSuite(t *testing.T) {
	suite.Run(t, new(FunctionDataCmdTestSuite))
}

type FunctionDataCmdTestSuite struct {
	suite.Suite
	function model.FunctionType
	data     *model.DeviceClassificationManufacturerDataType
	sut      *FunctionDataCmdImpl[model.DeviceClassificationManufacturerDataType]
}

func (suite *FunctionDataCmdTestSuite) SetupSuite() {
	suite.function = model.FunctionTypeDeviceClassificationManufacturerData
	suite.data = &model.DeviceClassificationManufacturerDataType{
		DeviceName: util.Ptr(model.DeviceClassificationStringType("device name")),
	}
	suite.sut = NewFunctionDataCmd[model.DeviceClassificationManufacturerDataType](suite.function)
	suite.sut.UpdateData(suite.data, nil, nil)
}

func (suite *FunctionDataCmdTestSuite) TestFunctionDataCmd_ReadCmd() {
	readCmd := suite.sut.ReadCmdType(nil, nil)
	assert.NotNil(suite.T(), readCmd.DeviceClassificationManufacturerData)
	assert.Nil(suite.T(), readCmd.DeviceClassificationManufacturerData.DeviceName)
	// TODO: assert on json
}

func (suite *FunctionDataCmdTestSuite) TestFunctionDataCmd_ReplyCmd() {
	readCmd := suite.sut.ReplyCmdType(false)
	assert.NotNil(suite.T(), readCmd.DeviceClassificationManufacturerData)
	assert.Equal(suite.T(), suite.data.DeviceName, readCmd.DeviceClassificationManufacturerData.DeviceName)
	// TODO: assert on json
}

// TODO: test NotifyCmdType
