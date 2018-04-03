
pragma solidity ^0.4.17;

import "./libraries/SafeMath.sol";

contract PriorityQueue {
    uint256[] private priorities;

    function PriorityQueue() {}

    function add(uint256 priority) {
        assert(priority != SafeMath.max());
        uint256 length = SafeMath.add(priorities.length, 1);
        priorities[length] = priority;
        bubbleUp();
    }

    function bubbleUp() {
        uint256[] storage p = priorities;

        uint256 i = SafeMath.sub(p.length, 1);

        while (i >= 0) {
            // Parent
            uint256 j = SafeMath.div(i, 2);

            if (p[j] < p[i]) {
                uint256 tmp = p[i];
                p[i] = p[j];
                p[j] = tmp;
            }
            else {
                break;
            }

            i = j;
        }
    }

    function remove(uint256 id) returns (bool) {
        // traverse through
        // BFS to index, and then bubble down.
        uint256[] storage p = priorities;
        uint256 i = 0;
        while (p[i] != id && i < priorities.length) {
            i++;
        }

        // We didn't find a match.
        if (i >= priorities.length) {
            return false;
        }

        p[i] = SafeMath.max();
        bubbleDown(i);
        return true;
    }

    function pop() returns (uint256) {
        uint256[] storage p = priorities;

        if (p.length == 0) {
            return SafeMath.max();
        }

        uint256 res = p[0];
        p[0] = SafeMath.max();
        bubbleDown(0);
        return res;
    }

    function bubbleDown(uint256 i) {
        uint256[] storage p = priorities;

        while(i < p.length) {
            uint256 j = SafeMath.mul(i, 2);
            uint256 k = SafeMath.add(SafeMath.mul(i, 2), 1);
            uint256 parent = p[i];
            uint256 left = p[j];
            uint256 right = p[k];
            if (left < right) {
                p[i] = left;
                p[j] = parent;
                // Move i to the sibling we chose.
                i = j;
            }
            else {
                // If we're equal and both are maxes
                // Then we move right which makes the
                // maxes right heavy.
                // Which is what we want.
                p[i] = right;
                p[k] = parent;
                // Move i to the sibling we chose.
                i = k;
            }
        }

        prune();
    }

    function prune() {
        uint256[] storage p = priorities;
        for(uint256 i = SafeMath.sub(p.length, 1); i >= 0; i--) {
            if(p[i] != SafeMath.max()){
                // reset length and break.
                p.length = SafeMath.add(i, 1);
                break;
            }
        }
    }
}