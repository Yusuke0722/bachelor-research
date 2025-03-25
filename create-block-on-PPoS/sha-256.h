/*
 * sha-256.h
 *
 *  Created on: Nov 12, 2022
 *      Author: ysk722
 */

#ifndef SHA_256_H_
#define SHA_256_H_

#include <stdio.h>
#include <stdlib.h>

void calc_sha_256(uint8_t hash[32], const void *input, size_t len);

#endif /* SHA_256_H_ */
