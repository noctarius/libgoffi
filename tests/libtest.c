/*
 * libgoffi - libffi adapter library for Go
 * Copyright 2019 clevabit GmbH
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include <stdlib.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#include <math.h>

extern int _sint() {
    return -1;
}

extern int8_t _sint8() {
    return -8;
}

extern int16_t _sint16() {
    return -16;
}

extern int32_t _sint32() {
    return -32;
}

extern int64_t _sint64() {
    return -64;
}

extern unsigned int _uint() {
    return 1;
}

extern uint8_t _uint8() {
    return 8;
}

extern uint16_t _uint16() {
    return 16;
}

extern uint32_t _uint32() {
    return 32;
}

extern uint64_t _uint64() {
    return 64;
}

extern float _float() {
    return 32.1;
}

extern double _double() {
    return -64.2;
}

extern double _sqrt(double v) {
    return sqrt(v);
}

extern int __sint(int v) {
    return v - 1;
}

extern int8_t __sint8(int8_t v) {
    return v - 8;
}

extern int16_t __sint16(int16_t v) {
    return v - 16;
}

extern int32_t __sint32(int32_t v) {
    return v - 32;
}

extern int64_t __sint64(int64_t v) {
    return v - 64;
}

extern unsigned int __uint(unsigned int v) {
    return v - 1;
}

extern uint8_t __uint8(uint8_t v) {
    return v - 8;
}

extern uint16_t __uint16(uint16_t v) {
    return v - 16;
}

extern uint32_t __uint32(uint32_t v) {
    return v - 32;
}

extern uint64_t __uint64(uint64_t v) {
    return v - 64;
}

extern float __float(float v) {
    return v - 32.;
}

extern double __double(double v) {
    return v - 64.;
}

extern const char *_char(const char *v, int length) {
    char *r = (char *)malloc(length);
    memcpy(r, v, length);
    return r;
}
