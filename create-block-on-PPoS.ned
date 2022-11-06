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

package create_block_on_ppos;

//
// TODO auto-generated type
//
network Create_block_on_PPoS
{
    types:
        simple Create
        {
            parameters:
                volatile double creationTime1 @unit(s) = triang(4s, 10s, 30.4s);
                volatile double creationTime2 @unit(s) = triang(30.4s, 55s, 80s);
        }

    submodules:
        Creator: Create;
}
