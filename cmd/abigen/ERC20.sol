
pragma solidity ^0.6.0;
// SPDX-License-Identifier: MIT

interface ERC20{
  function balanceOf(address) external view returns (uint256);
  function distroy(address _owner,uint256 _value) external;
  function symbol() view external  returns (string memory symbol_);
  function allowance(address _owner, address _spender) external view returns (uint256 remaining);
  function transferFrom(address, address, uint256) external returns (bool);
  function transfer(address _to, uint256 _value) external returns (bool success);
}
