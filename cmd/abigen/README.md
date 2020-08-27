

## true 测试网 V5

* orderMain 合约地址：  0xa27e2d16BfF7149DFc96e240098F31bde9777aC0
* fee 地址 0xA596bDa0Ea79C9478aB0071fc91D1Fb02A3a57e8
* 管理员地址： 0xA596bDa0Ea79C9478aB0071fc91D1Fb02A3a57e8
* 测试币地址： 0x66CC27601981B6DE5e51139ac0fED516782DE114

## true 测试网 V3

* orderMain 合约地址：  0x603E57DEfc197223E1F98Ec7006bf9eAB13a27E1
* fee 地址 0x6bE9780954580FCC268944e9D6271B3Dfc886997
* 管理员地址： 0xA596bDa0Ea79C9478aB0071fc91D1Fb02A3a57e8
* 测试币地址： 0x66CC27601981B6DE5e51139ac0fED516782DE114
## true 测试网 V3

* orderMain 合约地址：  0xC6C865fAdab0498e53014dC57dFfC51c2D15BD28
* fee 地址 0xA596bDa0Ea79C9478aB0071fc91D1Fb02A3a57e8
* 管理员地址： 0xA596bDa0Ea79C9478aB0071fc91D1Fb02A3a57e8
* 测试币地址： 0x66CC27601981B6DE5e51139ac0fED516782DE114
## true 测试网 V2

* orderMain 合约地址：  0x0fd2564A345fc6a124171A80cc5ab902fc3abc2e
* fee 地址 0xa9A2CbA5d5d16DE370375B42662F3272279B2b89
* 管理员地址： 0xA596bDa0Ea79C9478aB0071fc91D1Fb02A3a57e8

## true 测试网 V1

* orderMain 合约地址：  0x73B44bB8bE4ce6A7B1C1c2f3695b444C19F8f384
* 管理员地址： 0xA596bDa0Ea79C9478aB0071fc91D1Fb02A3a57e8
* 管理员私钥：0x255578b555d60595090845be3ef47e6e50bc00767b98c2fcb393487b28fc612d


* 测试代币 USDT 0x94E3cCe6F1716163a3E2f1E39D3cE3b4215C7eFC

* 合约部署地址：0xA596bDa0Ea79C9478aB0071fc91D1Fb02A3a57e8

  

#Defi contracts interface 
## orderMain.sol
### 1、创建supply 订单
  * Method: `function createSupOrder(
        uint8 _type,
        uint256 _id,
        address _orderOwner,
        address _coinAddr,
        uint256 _amount,
        uint256 _timeLevel,
        uint256 _rate,
        uint256 _createTime,
        uint256 _pledgeAmount
    ) public onlyManager`
  * Arguments:  
    * `_id`: 
    * `_orderOwner`: 
    * `_coinAddr`: 
    * `_amount`: 
    * `_timeLevel`: 
    * `_rate`: rateAmount
    * `_createTime`: 
    * `_pledgeAmount`: 

### 2、创建 loan 订单
  * function createLoanOrder(
        uint8 _type,
        uint256 _id,
        address _orderOwner,
        //address _pledgeCoin,
        uint256 _pledgeAmount,
        address _loanCoin,
        uint256 _loanAmount,
        uint256 _timeLevel,
        uint256 _rate,
        uint256 _fee,
        uint256 _compClosPrice,
        uint256 _createTime
        ) public onlyManager`
  * Arguments: 
    * `_type`: 
    * `_id`: 
    * `_pledgeCoin`: 
    * `_pledgeCoin`: 
    * `_pledgeAmount`: 
    * `_loanCoin`: 
    * `_loanAmount`: 
    * `_timeLevel`: 
    * `_rate`:rateAmount
    * `_fee`: 
    * `_compClosPrice`: 
    * `_createTime`: 

### 3、创建 match 撮合 订单
  * `function createMatchOrder(
        uint256 _id,
        uint256 _loanId,
        uint256[] memory _supIdList,
        uint256[] memory _sonSupIdList,
        uint256[] memory _amounts,
        uint256[] memory _rates,
        uint256[] memory _pledgeAmounts,
        uint256 _createtime//`
  * Arguments: 
    * `_loanId`: 
    * `_supIdList`: 
    * `_sonSupIdList`: 
    * `_amounts`:  这个数量表示一致
    * `_rates`: 
    * `_pledgeAmounts`: 
    * `_createtime`: 

    
   
### 4、处理 match  订单
  * `function processOrder(uint256 _moid,uint256 _time,uint256 _procType) public onlyManager{`
  * Arguments: 
    * `_moid`: 撮合单ID 
    * `_time`: 时间
    * `_procType`: 什么类型的处理，正常，逾期，还是爆仓

### 5、提现订单
  * `withdraw(uint256 _supId) public`
  * Arguments: 
    * `_supId`: 挂单ID

### 6、系统充值USDT
* `function RechargeCoin(address _coinAddr,address _fromAddr,uint256 _amount) public onlyOwner`

 * Arguments: 
    * `_coinAddr`: USDT地址
    *  `_fromAddr`: 从哪个地址转出
    *  `_amount`: 数量




 
