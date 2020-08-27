pragma solidity ^0.6.0;
// SPDX-License-Identifier: MIT
library orderStruct{
    
    enum soState {soNULL,soPending,soSuppingAll,soSuppingPart,soSuppingEnd}
    enum lo_moState {loNULL,loPending,loLoaning,loRepayment,soOverPrice,soOverTime}
    
    
     //supply Order
    struct supplyOrd{
        uint256 so_id;  // apply order id ,create by manarge
        address so_owner; // who send this order
        address so_coinAddr; // what kind of coin 
        uint256 so_amount;   // this is order amount 
        uint256 so_timelevel;   //  loan time level 7 or 15 days
        uint256 so_rateAmount;   // the rate  for time level
        soState so_state;  // order ao_state
        uint256 so_time; // this is order create time 
        uint256[] so_sonApplyOrd;
        uint256 so_activeAmount;
        uint256 so_finshTime; // this is finsh time 
        uint256 so_pledgeAmount;
    }
    
    // son apply order 
    struct sonSpplyOrd {
        uint256 sonso_id;
        uint256 sonso_parentId; // parent order id 
        uint256 sonso_loanId; // loan order id 
        uint256 sonso_amount; // loan amount 
        uint256 sonso_time;  // when create this order 
        uint256 sonso_rateAmount; // this order can win coin
        soState sonso_state;
        uint256 sonso_finshTime; // this is finsh time
        uint256 sonso_pledgeAmount;
    }
    
    // loan order 
    struct loanOrd{
        uint256 lo_id; // this is loan order id 
        address lo_owner;// how send this order 
        //address lo_pledgeCoin; // what kinds of coin  for pledge
        uint256 lo_pledgeAmount; // how many for pledge
        address lo_loanCoin; // what kinds of coin for loan
        uint256 lo_loanAmount; // how many for loan
        uint256 lo_timeLevel; // how long to  loan
        uint256 lo_rateAmount;  // how many need pay
        uint256 lo_feeAmount;  // this fee is for 
        uint256 lo_compClosePrice ; // compulsoryClosingPrice  this price need *1000
        lo_moState lo_state; // this is state 
        uint256 lo_time; // this is order create time 
        uint256 lo_finshTime; // this is finsh time 
    }
    
    struct matchOrd{
        uint256 mo_id; // this is mathc order id,create by manarge
        uint256 mo_loId; // loan order  id 
        uint256[] mo_soId; // this is list supply id 
        uint256[] mo_sonSoId; // if there hava son id need create a son id 
        uint256[] mo_amount; // this is ether id amount 
        uint256[] mo_rateAmount; // this win how many 
        uint256 mo_time;  // this is create time 
        lo_moState  mo_state; // this is order state 
        uint256 mo_finshTime; // this is finsh time 
    }
    
}