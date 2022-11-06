//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
// 
// You should have received a copy of the GNU Lesser General Public License
// along with this program.  If not, see http://www.gnu.org/licenses/.
// 

package create_block_on_pow;

//
// TODO auto-generated type
//
network Create_block_on_PoW
{
    types:
        channel Channel extends ned.DelayChannel {
            delay = 100ms;
        }
        simple Create
        {
            parameters:
                volatile double creationTime @unit(s) = gamma_d(1.10, 34.12s);
            gates:
                inout gate[];
        }

    submodules:
        Creator[100]: Create;
    connections:
        for i=0..sizeof(Creator)-1, for j=0..sizeof(Creator)-1 {
            Creator[i].gate++ <--> Channel <--> Creator[j].gate++ if i<j;
        }
}
