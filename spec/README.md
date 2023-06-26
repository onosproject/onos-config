<!--
SPDX-FileCopyrightText: 2023-present Intel Corporation

SPDX-License-Identifier: Apache-2.0
-->

# TLA+ Specification for µONOS Config

This directory contains a complete [TLA+](https://lamport.azurewebsites.net/tla/tla.html) specification and
model for the updated µONOS Config control plane logic. The spec is checked with TLC, using a variety of safety and
liveness properties to verify the design can recover from failures and does not contain edge cases deadlock/livelock.
The model is also capable of outputting test cases which can be used to verify the implementation of the spec.
