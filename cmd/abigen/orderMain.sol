pragma solidity ^0.6.0;
// SPDX-License-Identifier: MIT

import "./SafeMath.sol";
import "./orderStruct.sol";
import "./ERC20.sol";


contract orderMain{
    
    using SafeMath for *;
    address public owner; // 这个是合约部署者
    address public feeAddr;
   // address public sysAddr;
    mapping(address=>bool) public managers;
    
    // balance 
    address constant trueAddr = address(0x000000000000000000000000000000000000000F); //
    mapping(address => uint256) public totalBalance; // coin addr => balance different this is usdt  存储所有的代币信息
    mapping(address => uint256) public plyPledgeBalance; // 抵押代币的数量，coinAddr =》 balance  用于抵押的时候计数

    //order
    mapping(uint256 => orderStruct.supplyOrd) public sup_ordList; // supply order list   orderID => order
    mapping(uint256 => orderStruct.sonSpplyOrd) public sons_ordList; // son supply order list orderID => order 
    mapping(uint256 => orderStruct.loanOrd) public loan_ordList;  // loan order list  orderID => order
    mapping(uint256 => orderStruct.matchOrd) public match_ordList; // match order list */
    
    constructor(address _fee/*address _sysAddr*/) public{
        owner = msg.sender;
        //sysAddr = _sysAddr;
        feeAddr = _fee;
    }
    
    fallback() external payable{
        
    }
    receive() external payable{
        totalBalance[trueAddr]  = totalBalance[trueAddr].add(msg.value);
        plyPledgeBalance[msg.sender] = plyPledgeBalance[msg.sender].add(msg.value);
    }
    
    // create 创建 供应单，创建供应单之前，用户需要先Approve TUSDT让合约转
    function createSupOrder(
        uint256 _id,
        address _orderOwner,
        address _coinAddr,
        uint256 _amount,
        uint256 _timeLevel,
        uint256 _createTime
    ) public onlyManager{
        
        // check the id 
        // 检查供应单
        require(checkSupOrder(_id,_coinAddr),"check suplorder false");
        
        //get suppler token by contract 
        // 先把供应单的钱转到合约
        uint256 beforBalance = ERC20(_coinAddr).balanceOf(address(this));
        ERC20(_coinAddr).transferFrom(_orderOwner,address(this),_amount);
        uint256 afterBalance = ERC20(_coinAddr).balanceOf(address(this));
        require(beforBalance == afterBalance.sub(_amount),"transfer ERC20 supply error");
        //供应单的钱会进入 totalBalance
        totalBalance[_coinAddr] = totalBalance[_coinAddr].add(_amount);
        
        //创建供应单
        uint256[] memory sonID;
        orderStruct.supplyOrd memory sOrder = orderStruct.supplyOrd(
            _id,
            _orderOwner,
            _coinAddr,
            _amount,
            _timeLevel,
            0,
            orderStruct.soState.soPending,
            _createTime,
            sonID,
            _amount,
            0,
            0);
            
        sup_ordList[_id] = sOrder;

    }
    
    //创建 
    function createLoanOrder(
        uint256 _id,
        address _orderOwner,
        uint256 _pledgeAmount,
        address _loanCoin,
        uint256 _loanAmount,
        uint256 _timeLevel,
        uint256 _rateAmount,
        uint256 _fee,
        uint256 _compClosPrice,
        uint256 _createTime
        ) public onlyManager{
        
        //先坚持 借单是不是存在
        require(checkLoanOrder(_id,_loanCoin,_loanAmount,_pledgeAmount,_orderOwner),"check loan order false");
        
        //创建 借单 
         orderStruct.loanOrd memory loanOrder = orderStruct.loanOrd(
            _id,
            _orderOwner,
            _pledgeAmount,
            _loanCoin,
            _loanAmount,
            _timeLevel,
            _rateAmount,
            _fee,
            _compClosPrice,
            orderStruct.lo_moState.loPending,
            _createTime,
            _createTime);
            
        loan_ordList[_id] = loanOrder;
        
        //to storage the pledge amount when ply storage the true to this contract 
        // 钱已经进入合约，删除这个计数
        plyPledgeBalance[_orderOwner] = plyPledgeBalance[_orderOwner].sub(_pledgeAmount);
        
    }

    //创建撮合单，这个单是再借单
    function createMatchOrder(
        uint256 _id,
        uint256 _loanId,
        uint256[] memory _supIdList,
        uint256[] memory _sonSupIdList,
        uint256[] memory _amounts,
        uint256[] memory _rateAmounts,
        uint256[] memory _pledgeAmounts,
        uint256 _createtime//
        ) public onlyManager{
            uint256 len = _supIdList.length;
            require(len<16,"too length");
            
            //检查 撮合单
            require(checkMatchOrder(_id,_loanId,_supIdList,_sonSupIdList,_amounts,_rateAmounts),"checkt match order fail");
            
            //创建撮合单
            orderStruct.matchOrd memory mOrder = orderStruct.matchOrd(
                _id,
                _loanId,
                _supIdList,
                _sonSupIdList,
                _amounts,
                _rateAmounts,
                _createtime,
                orderStruct.lo_moState.loPending,
                _createtime);
                
            match_ordList[_id] = mOrder;
            
            // set loandID state
            // 更新借款单状态
            loan_ordList[_loanId].lo_state = orderStruct.lo_moState.loLoaning;
            
            
            address to;
            uint256 rateAmount;
             //send loan token
            orderStruct.loanOrd storage loOrder =   loan_ordList[_loanId];
            
            //totalbalance 
            //所有借款的款项都要分出去
            totalBalance[loOrder.lo_loanCoin] = totalBalance[loOrder.lo_loanCoin].sub(loOrder.lo_loanAmount);
            
            for(uint256 i=0;i<len;i++){
                
                to = sup_ordList[_supIdList[i]].so_owner;
                rateAmount = _rateAmounts[i];
                
                if(_sonSupIdList[i] == 0){
                    sup_ordList[_supIdList[i]].so_state = orderStruct.soState.soSuppingAll;
                    sup_ordList[_supIdList[i]].so_activeAmount = 0;
                    
                    //send rate Amount to supply custome
                    //分的是 利息
                    transferCoin(loOrder.lo_loanCoin,to,rateAmount,true);
                    sup_ordList[_supIdList[i]].so_pledgeAmount = _pledgeAmounts[i];
                    sup_ordList[_supIdList[i]].so_rateAmount = _rateAmounts[i];
                    
                }else{
                    //更新状态和余额
                    sup_ordList[_supIdList[i]].so_state = orderStruct.soState.soSuppingPart; 
                    sup_ordList[_supIdList[i]].so_activeAmount =  sup_ordList[_supIdList[i]].so_activeAmount.sub(_amounts[i]);
                    sup_ordList[_supIdList[i]].so_sonApplyOrd.push(_sonSupIdList[i]);
                    //创建子单
                    createSonSupOrder(_sonSupIdList[i],_supIdList[i],_loanId,_amounts[i],_rateAmounts[i],_pledgeAmounts[i]);
                    
                    //send rate Amount to supply custome
                    //分的是 利息
                    transferCoin(loOrder.lo_loanCoin,to,rateAmount,true);
                }
            }
            
           
            uint256 leftAmount = loOrder.lo_feeAmount.add(loOrder.lo_rateAmount); // fee + rate
            uint256 loanAmount = loOrder.lo_loanAmount.sub(leftAmount); // loan usdt amount 给借方发的钱，实际
            
            // 分钱给借方
            transferCoin(loOrder.lo_loanCoin,loOrder.lo_owner,loanAmount,true);
            
            
            //send  fee
            // 分手续费
            transferCoin(loOrder.lo_loanCoin,feeAddr,loOrder.lo_feeAmount,true);
            
            
            
            
    }
    
    //提现，只能是管理员才可以
    function withdraw(uint256 _supId) public onlyManager {
        
        uint256 amount = sup_ordList[_supId].so_activeAmount;
        
        require( amount >0,"not enght balance");
        address coinAddr = sup_ordList[_supId].so_coinAddr;
        address to = sup_ordList[_supId].so_owner;
        
        sup_ordList[_supId].so_activeAmount = 0;
        sup_ordList[_supId].so_state = orderStruct.soState.soSuppingEnd;
        
        transferCoin(coinAddr,to,amount,true);
         
    }
    
    
    /*function rechargeCoin(address _coinAddr,address _from,uint256 _amount) public onlyManager{
        
        require(sysAddr == _from,"from not right");
        uint256 beforBalance = ERC20(_coinAddr).balanceOf(address(this));
        ERC20(_coinAddr).transferFrom(_from,address(this),_amount);
        uint256 afterBalance = ERC20(_coinAddr).balanceOf(address(this));
        require(beforBalance == afterBalance.sub(_amount),"transfer ERC20 supply error");
        
        systemBalance[_coinAddr] = systemBalance[_coinAddr].add(_amount);
    }
    
    function withdrawSysBalance(address _coinAddr,uint256 _amount) public onlyManager{
        uint256 toAmount;
        if (systemBalance[_coinAddr] >= _amount){
            systemBalance[_coinAddr] = systemBalance[_coinAddr].sub(_amount);
            toAmount = _amount;
        }else{
            toAmount = systemBalance[_coinAddr];
            systemBalance[_coinAddr] = 0;
            
        }
        transferCoin(_coinAddr,sysAddr,toAmount,true);
    }*/
    
    function processOrder(uint256 _moid,uint256 _time,uint256 _procType) public onlyManager{
        
        require(_procType <=2,"process type error");
        
    
        if (_procType == 1){ // match order nomall end 
            procOrderNormal(_moid,_time);
        }else if(_procType == 2){ // match order over time 
            //procOrderOverTime(_moid,_time);
        /*}else if(_procType == 3){ // match order over the price*/
            procOrderOverPrice(_moid,_time);
        }
        
    }
    
    //正常执行 这个撮合单 需要借款人把单先放到这边
    function procOrderNormal(uint256 _moid,uint256 _time) internal{
        
        orderStruct.matchOrd storage moOrder =   match_ordList[_moid];
        orderStruct.loanOrd storage loOrder =   loan_ordList[moOrder.mo_loId];
        address coinAddr = loOrder.lo_loanCoin;
        
        //先把钱转转到合约
        uint256 beforBalance = ERC20(coinAddr).balanceOf(address(this));
        ERC20(coinAddr).transferFrom(loOrder.lo_owner,address(this),loOrder.lo_loanAmount);
        uint256 afterBalance = ERC20(coinAddr).balanceOf(address(this));
        require(beforBalance == afterBalance.sub(loOrder.lo_loanAmount),"transfer ERC20 procOrderNormal error");
        
        //totalBalance[coinAddr] = totalBalance[coinAddr].sub(toAmount);
        uint256 totalAmount = loOrder.lo_loanAmount;
        
        // order del
        //订单进行处理
        procUpdate_lo_moOrder(_moid,_time,orderStruct.lo_moState.loRepayment,orderStruct.soState.soSuppingEnd);
        
        uint256 len = moOrder.mo_soId.length;
        address to;
        uint256 toAmount;
        
        //send usdt to supply player 
        for(uint256 i=0;i<len;i++){
            
            if (moOrder.mo_sonSoId[i] != 0){
                uint256 parentID = sons_ordList[moOrder.mo_sonSoId[i]].sonso_parentId;
                to = sup_ordList[parentID].so_owner;
                toAmount = sons_ordList[moOrder.mo_sonSoId[i]].sonso_amount;
                
                require(totalAmount>=toAmount,"not enght systemBalance");
                totalAmount = totalAmount.sub(toAmount);
                
                // 转本金给借方
                transferCoin(coinAddr,to,toAmount,true);
                
               
            }else{
                //if (sup_ordList[moOrder.mo_soId[i]].so_state == orderStruct.soState.soSuppingAll){
                to = sup_ordList[moOrder.mo_soId[i]].so_owner;
                toAmount =  sup_ordList[moOrder.mo_soId[i]].so_amount;
                    
                require(totalAmount>=toAmount,"not enght systemBalance");
                totalAmount = totalAmount.sub(toAmount);
                     // 转本金给借方
                transferCoin(coinAddr,to,toAmount,true);
                   
                //}    
            }
            
        }
        
        //back the true to loan player 
        to = loOrder.lo_owner;
        toAmount = loOrder.lo_pledgeAmount;
        //还true 给 借款放
        totalBalance[trueAddr] = totalBalance[trueAddr].sub(toAmount);
        transferCoin(trueAddr,to,toAmount,false);
        
        
    }
    
    /*function procOrderOverTime(uint256 _moid,uint256 _time) internal{
        //send system usdt to supply player 
        procUpdate_lo_moOrder(_moid,_time,orderStruct.lo_moState.soOverTime,orderStruct.soState.soSuppingEnd);
        
        //send usdt to supply player 
        orderStruct.matchOrd storage moOrder =   match_ordList[_moid];
        orderStruct.loanOrd storage loOrder =   loan_ordList[moOrder.mo_loId]; 
         
        uint256 len = moOrder.mo_soId.length;
        address  to;
        uint256 toAmount;
        address coinAddr = loOrder.lo_loanCoin;
        
        //send usdt to supply player 
        for(uint256 i=0; i<len; i++){
            
            if (moOrder.mo_sonSoId[i] != 0){//如果 儿子订单不为空，就按儿子订单数量还
                //if(sons_ordList[moOrder.mo_sonSoId[i]].so)
                uint256 parentID = sons_ordList[moOrder.mo_sonSoId[i]].sonso_parentId;
                to = sup_ordList[parentID].so_owner;
                toAmount = sons_ordList[moOrder.mo_sonSoId[i]].sonso_amount;
                
                require(systemBalance[coinAddr]>=toAmount,"not enght systemBalance");
                systemBalance[coinAddr] = systemBalance[coinAddr].sub(toAmount);
                
                transferCoin(coinAddr,to,toAmount,true);
                
                
            }else{
                
                    //如果是父订单，就按父订单来处理
                to = sup_ordList[moOrder.mo_soId[i]].so_owner;
                toAmount = sup_ordList[moOrder.mo_soId[i]].so_amount;
                    
                require(systemBalance[coinAddr]>=toAmount,"not enght systemBalance");
                systemBalance[coinAddr] = totalBalance[coinAddr].sub(toAmount);
                    
                transferCoin(coinAddr,to,toAmount,true);
                    
            }
            
        }
        
        //发送 true 给fee 地址
        transferCoin(coinAddr,feeAddr,loOrder.lo_pledgeAmount,false);
        
        
    }*/
    
    //逾期 
    function procOrderOverPrice(uint256 _moid,uint256 _time) internal{
        //send system true to supply player
        //修改订单状态
       procUpdate_lo_moOrder(_moid,_time,orderStruct.lo_moState.soOverPrice,orderStruct.soState.soSuppingEnd);
       
       //send usdt to supply player 
        orderStruct.matchOrd storage moOrder =   match_ordList[_moid];
        //orderStruct.loanOrd storage loOrder =   loan_ordList[moOrder.mo_loId];
        
        uint256 len = moOrder.mo_soId.length;
        address  to;
        uint256 toAmount;
      
        totalBalance[trueAddr] = totalBalance[trueAddr].sub(loan_ordList[moOrder.mo_loId].lo_pledgeAmount);
        //send usdt to supply player 
        for(uint256 i=0;i<len;i++){
            
            if (moOrder.mo_sonSoId[i] != 0){
                uint256 parentID = sons_ordList[moOrder.mo_sonSoId[i]].sonso_parentId;
                to = sup_ordList[parentID].so_owner;
                toAmount = sons_ordList[moOrder.mo_sonSoId[i]].sonso_pledgeAmount;
                //payable(to).transfer(toAmount);
                //发送true 给 借方
                transferCoin(trueAddr,to,toAmount,false);
                
            }else{
                if (sup_ordList[moOrder.mo_soId[i]].so_state == orderStruct.soState.soSuppingAll){
                    to = sup_ordList[moOrder.mo_soId[i]].so_owner;
                    toAmount = sup_ordList[moOrder.mo_soId[i]].so_pledgeAmount;
                    
                    //发送true 给 借方
                    transferCoin(trueAddr,to,toAmount,false);
                  
                }    
            }
            
        }
        
    }
    
    //修改订单状态 再执行撮合单的时候
    function procUpdate_lo_moOrder(
        uint256 _moid,
        uint256 _time,
        orderStruct.lo_moState _lomostate,
        orderStruct.soState _soState) internal{
        
        //match order 
        orderStruct.matchOrd storage moOrder =   match_ordList[_moid];
        moOrder.mo_state = _lomostate;
        moOrder.mo_finshTime = _time;
        
        //loan order 
        loan_ordList[moOrder.mo_loId].lo_state = _lomostate;
        loan_ordList[moOrder.mo_loId].lo_finshTime = _time;
        
        // supply order 
        uint256 len = moOrder.mo_soId.length;
        for(uint256 i=0;i<len;i++){
            if (moOrder.mo_sonSoId[i] != 0){
                sons_ordList[moOrder.mo_sonSoId[i]].sonso_state = _soState;
                sons_ordList[moOrder.mo_sonSoId[i]].sonso_finshTime = _time;
            }else{
                if (sup_ordList[moOrder.mo_soId[i]].so_state == orderStruct.soState.soSuppingAll){
                    sup_ordList[moOrder.mo_soId[i]].so_state = _soState;
                    sup_ordList[moOrder.mo_soId[i]].so_finshTime = _time;
                }
                
            }
        }
    }

    // 创建子单
    function createSonSupOrder(
        uint256 _id,
        uint256 _parentId,
        uint256 _loanId,
        uint256 _amount,
        uint256 _rate,
        uint256 _pledgeAmount
        ) internal{
            
            orderStruct.sonSpplyOrd memory sonOrder = orderStruct.sonSpplyOrd(
                 _id,
                 _parentId,
                 _loanId,
                 _amount,
                 now,
                 _rate,
                 orderStruct.soState.soSuppingAll,
                 now,
                 _pledgeAmount);
                 
            sons_ordList[_id] = sonOrder;
        
    }
    
    // 检查撮合单
    function checkMatchOrder(
        uint256 _id,
        uint256 _loanId,
        uint256[] memory _supIdList,
        uint256[] memory _sonSupIdList,
        uint256[] memory _amounts,
        uint256[] memory _rates) internal view returns(bool){
            // check the match order
            // 撮和单应该是空的
            if(match_ordList[_id].mo_state != orderStruct.lo_moState.loNULL){
                return false;
            }
            
            //check loan order state
            // 借单应该挂着
            if(loan_ordList[_loanId].lo_state != orderStruct.lo_moState.loPending){
                return false;
            }
            // 所有金额必须大于等于 借款金额
            if(totalBalance[loan_ordList[_loanId].lo_loanCoin] < loan_ordList[_loanId].lo_loanAmount){
                return false;
            }
            
            //check supply order 
            uint256 len = _supIdList.length;
            uint256 totalRate;
            uint256 totalAmount;
            // 撮合单信息是不是对的
            if(len != _sonSupIdList.length || len != _amounts.length || len != _rates.length){
                return false;
            }
            
            //check all supply order state and active amount
            for(uint256 i=0;i<len;i++){
                // 先判断父单是不是对的
                if(sup_ordList[_supIdList[i]].so_state != orderStruct.soState.soPending &&  sup_ordList[_supIdList[i]].so_state != orderStruct.soState.soSuppingPart){
                    return false;
                }
                //所有的借款单和
                totalRate = totalRate.add(_rates[i]);
                totalAmount = totalAmount.add(_amounts[i]);
                
                
                if(_sonSupIdList[i] == 0){
                    // 如果父单有儿子单，就必须要用子单
                    if (sup_ordList[_supIdList[i]].so_sonApplyOrd.length !=0){
                        return false;
                    }
                    //如果是所有的借出就判断数量是不是一致
                    if(_amounts[i] != sup_ordList[_supIdList[i]].so_amount){
                        return false;
                    }
                    
                }else{
                    //子单必须不存在
                    if(sons_ordList[_sonSupIdList[i]].sonso_state != orderStruct.soState.soNULL){
                        return false;
                    }
                    //子单借的数量必须小于父单的可借数量
                    if(_amounts[i] > sup_ordList[_supIdList[i]].so_activeAmount){
                        return false;
                    }
                }
            }
             
            //check total rate //check amount
            if (totalRate != loan_ordList[_loanId].lo_rateAmount || totalAmount != loan_ordList[_loanId].lo_loanAmount){
                return false;
            }
            
           return true;
        
    }
    
    //转账
    function transferCoin(address _coinAddr,address _to,uint256 _amount,bool isContractCoin) internal {
        uint256 beforBalance;
        uint256 afterBalance;
        
        if (isContractCoin){
            beforBalance = ERC20(_coinAddr).balanceOf(_to);
            ERC20(_coinAddr).transfer(_to,_amount);
            afterBalance = ERC20(_coinAddr).balanceOf(_to);
        }else{
            beforBalance = address(_to).balance;
            payable(_to).transfer(_amount);
            afterBalance = address(_to).balance;
        }
        
        require(beforBalance == afterBalance.sub(_amount),"transfer transferCoin  error "); 
        
        
    }
    
    //check the loan  order make sure is right correct
    function checkLoanOrder(uint256 _id,address _loanCoin,uint256 _landAmount,uint256 _pledgeAmount,address _orderOwner) internal view returns(bool){
        
        //借款单是应该空的
        if(loan_ordList[_id].lo_state != orderStruct.lo_moState.loNULL){
            return false;
        }
        
        //uint256 wantBalance;
       // uint256 activeBalance;
       if (_landAmount > totalBalance[_loanCoin]){
           return false;
       }
        
        // 检查余额
       /* wantBalance = totalBalance[_coinAddr];
        activeBalance = address(this).balance;
        
        if (wantBalance != activeBalance){
            return false;
        }*/
        //检查借单的钱
        if(plyPledgeBalance[_orderOwner] < _pledgeAmount){
            return false;
        }
        return true;
    }
    
    //check the supply order make sure is correct
    function checkSupOrder(uint256 _id,address _coinAddr) internal view returns(bool){
        //check exit
        //正常订单的状态如果没有就是空
        if (sup_ordList[_id].so_state != orderStruct.soState.soNULL){
            return false;
        }
        
        //供应单是必须不是true
        if (_coinAddr == trueAddr){
            return false;
        }

        return true;
    }
    
    // this is add or del manager 
    function setManager(address _mAddr, bool isAdd) public onlyOwner{
        
        if (isAdd) {
            managers[_mAddr] = true;
        }else{
            managers[_mAddr] = false;
        }
        
    }
    
    modifier onlyOwner{
        require(msg.sender == owner,"only owner");
        _;
    }
    
    modifier onlyManager{
        require(managers[msg.sender],"only managers");
        _;
    }
    
}
