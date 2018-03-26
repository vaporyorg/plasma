pragma solidity ^0.4.17;

import './libraries/ByteUtils.sol';
import './libraries/RLP.sol';
import './libraries/SafeMath.sol';

contract Plasma {
    using SafeMath for uint256;
    using RLP for bytes;
    using RLP for RLP.RLPItem;
    using RLP for RLP.Iterator;

    event Deposit(address sender, uint value);
    event SubmitBlock(address sender, bytes32 root);
    event ExitStarted(address sender, uint amount, uint blocknum, uint txindex, uint oindex);
    event DebugBytes32(address sender, bytes32 item);
    event DebugBytes(address sender, bytes item);
    event DebugAddress(address sender, address item);
    event DebugUint(address sender, uint item);
    event DebugBool(address sender, bool item);

    address public authority;
    mapping(uint256 => childBlock) public childChain;
    mapping(uint256 => exit) public exits;
    uint256 public currentChildBlock;
    uint256[] public exitsIndexes;

    struct childBlock {
        bytes32 root;
        uint256 created_at;
    }

    struct exit {
        address owner;
        uint256 amount;
        uint256 blocknum;
        uint256 txindex;
        uint256 oindex;
        uint256 started_at;
    }

    function Plasma() {
        authority = msg.sender;
        currentChildBlock = 1;
    }

    function submitBlock(bytes32 root) public {
        require(msg.sender == authority);
        childChain[currentChildBlock] = childBlock({
            root: root,
            created_at: block.timestamp
        });
        currentChildBlock = currentChildBlock.add(1);

        SubmitBlock(msg.sender, root);
    }

    function deposit(bytes txBytes) public payable {
        var txItem = txBytes.toRLPItem();
        var txList = txItem.toList();

        // TODO: update after format of txBytes changes.
        require(msg.sender == txList[0].toAddress());

        bytes32 root = createSimpleMerkleRoot(txBytes);

        childChain[currentChildBlock] = childBlock({
            root: root,
            created_at: block.timestamp
        });

        currentChildBlock = currentChildBlock.add(1);

        Deposit(msg.sender, msg.value);
    }

    function createSimpleMerkleRoot(bytes txBytes) returns (bytes32) {
        bytes32 zeroBytes;
        
        // TODO: new bytes(130) must match how we hash empty nodes on side-chain.
        bytes32 root = keccak256(keccak256(txBytes), new bytes(130));
        for (uint i = 0; i < 16; i++) {
            root = keccak256(root, zeroBytes);
            zeroBytes = keccak256(zeroBytes, zeroBytes);
        }

        return root;
    }

    function startExit(
        uint256 blocknum,
        uint256 txindex,
        uint256 oindex,
        bytes txBytes,
        bytes proof
    ) public
    {
        var txItem = txBytes.toRLPItem();
        var txList = txItem.toList();

        // TODO: update after format of txBytes changes.
        var baseIndex = oindex * 2;

        require(msg.sender == txList[baseIndex].toAddress());

        var amount = txList[baseIndex + 1].toUint();

        var res = checkProof(blocknum, txindex, txBytes, proof);

        DebugBool(msg.sender, res);

        uint256 length = exitsIndexes.length++;
        exitsIndexes[length] = length;
        
        exits[length] = exit({
            owner: msg.sender,
            amount: amount,
            // These are necessary for challenges.
            blocknum: blocknum,
            txindex: txindex,
            oindex: oindex,
            started_at: block.timestamp
        });

        ExitStarted(msg.sender, amount, blocknum, txindex, oindex);
    }

    function checkProof(
        uint256 blocknum,
        uint256 txindex,
        bytes txBytes,
        bytes proof
    ) returns (bool)
    {
        // TODO: might need to adjust depth
        require(proof.length == 15 * 32);

        var root = childChain[blocknum].root;

        var otherRoot = keccak256(txBytes);

        // Offset for bytes assembly starts at 32
        uint j = 32;

        // TODO: might need to adjust depth
        for(uint i = 0; i < 15; i++) {
            bytes32 sibling;
            assembly {
                sibling := mload(add(proof, j))
            }
            j += 32;

            if (txindex % 2 == 0) {
                otherRoot = keccak256(otherRoot, sibling);
            } else {
                otherRoot = keccak256(sibling, otherRoot);
            }
            
            txindex = txindex / 2;
        }

        return otherRoot == root;
    }
}
