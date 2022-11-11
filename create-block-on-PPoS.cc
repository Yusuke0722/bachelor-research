/*
 * create-block-on-PoW.cc
 *
 *  Created on: Nov 6, 2022
 *      Author: ysk722
 */

#include <string>
#include <vector>
#include <stdio.h>
#include <omnetpp.h>
#include <nlohmann/json.hpp>
#include "sha-256.h"

#define BUFFER_MAX 4096
#define LLAMBDA 60.0
#define SLAMBDA 10.0
#define WAIT 1.0
#define DIF 25
#define STEP_MAX 180


using namespace omnetpp;
using namespace std;
using json = nlohmann::json;

struct block {
    uint32_t contents_length;
    uint8_t contents_hash[32];
    uint8_t previous_hash[32];
    uint32_t step;
    int sign;
};

class Create: public cSimpleModule {
private:
    cHistogram createTime;
    simtime_t miningTime;
    simtime_t stepTime;
    block newBlock;
    uint32_t round;
    int gate_size;
    int tH;
    int lead_sign;
    int bit;
    int seed;
    vector<json> messages;
protected:
    void initialize() override;
    void send_to_all(const string name);
    void handleMessage(cMessage *msg) override;
    void publish_message(uint32_t step, int sign);
    void step12();
    void find_leader();
    void mineBlock();
    bool is_leader(json message, vector<json> value, int num);
    void find_value(uint32_t step);
    bool is_reach_tH(json message, vector<json> valid);
    bool is_finalized();
    int  select_bit(uint32_t step);
    void coin_flipped(uint32_t step);
    void step();
    void finish() override;
};

Define_Module(Create);

void Create::initialize() {
    gate_size = gateSize("gate");
    tH = 0.69 * gate_size;
    scheduleAt(simTime(), new cMessage("create"));
    round = 0;
    seed = getId();
}

void Create::send_to_all(const string name) {
    for (int i = 0; i < gate_size; i++) {
        send(new cMessage(name.c_str()), "gate$o", i);
    }
}

void Create::handleMessage(cMessage *msg) {
    string str = string(msg->getName());
    if ("create" == str) { mineBlock();
    } else if ("leader" == str) { find_leader();
    } else if ("value" == str) { find_value(newBlock.step);
    } else if ("coin" == str) { coin_flipped(newBlock.step);
    } else if ("find" == str) {
        createTime.collect(simTime() - miningTime);
        miningTime = simTime();
    } else if ("step" == str) {
        newBlock.step++;
        if (is_finalized() || newBlock.step > STEP_MAX) {
            scheduleAt(simTime(), new cMessage("finish"));
        } else { step(); }
    } else if ("finish" == str) {
        scheduleAt(simTime(), new cMessage("create"));
        seed = newBlock.sign;
        round++;
        if (newBlock.step <= STEP_MAX && newBlock.sign == lead_sign) {
            createTime.collect(simTime() - miningTime);
            send_to_all("find");
        }
    } else { messages.push_back(json::parse(str)); }
    delete msg;
}

block buildBlock(const block *previous, const char *contents, uint64_t length) {
    block header;
    header.contents_length = length;
    if (previous) {
        /* calculate previous block header hash */
        calc_sha_256(header.previous_hash, previous, sizeof(block));
    } else {
        /* genesis has no previous. just use zeroed hash */
        memset(header.previous_hash, 0, sizeof(header.previous_hash));
    }
    /* add data hash */
    calc_sha_256(header.contents_hash, contents, length);
    return header;
}

void Create::publish_message(uint32_t step, int sign) {
    uint8_t selected[32];
    calc_sha_256(selected, &newBlock, sizeof(block));
    if (selected[0] < DIF) {
        json j;
        if (step == 1) {
            EV << "proposing block" << endl;
        }
        if (step < 4) {
            j = {{"round", round}, {"step", step}, {"sign", sign}};
        } else {
            j = {{"round", round}, {"step", step}, {"sign", sign}, {"bit", bit}};
        }
        messages.push_back(j);
        send_to_all(j.dump());
    }
}

void Create::step12() {
    newBlock.step = 1;
    publish_message(1, newBlock.sign);
    EV << "step2" << endl;
    newBlock.step++;
    lead_sign = RAND_MAX;
    stepTime = simTime();
    scheduleAt(simTime() + 2*SLAMBDA, new cMessage("leader"));
}

void Create::find_leader() {
    for (json m : messages) {
        if (m["round"] == round && m["step"] == 1 && m["sign"] < lead_sign) {
            lead_sign = m["sign"];
        }
    }

    if (lead_sign != RAND_MAX || simTime() - stepTime >= SLAMBDA + LLAMBDA) {
        publish_message(2, lead_sign);
        scheduleAt(simTime(), new cMessage("step"));
    } else {
        scheduleAt(simTime() + WAIT, new cMessage("leader"));
    }
}

void Create::mineBlock() {
    miningTime = simTime();
    char line_buffer[BUFFER_MAX] = "block";
    uint64_t size = strnlen(line_buffer, BUFFER_MAX) + 1;

    block *previous_ptr = NULL;
    newBlock = buildBlock(previous_ptr, line_buffer, size);
    srand(seed);
    newBlock.sign = rand();
    step12();
}

bool Create::is_leader(json message, vector<json> value, int num) {
    int count = 0;
    for (json v : value) {
        if (message["sign"] == v["sign"]) { count++; }
        if (count >= num) {
            bit = (num == tH && message["sign"] != RAND_MAX) ? 0 : 1;
            return true;
        }
    }
    bit = 1;
    return false;
}

void Create::find_value(uint32_t step) {
    vector<json> value;
    bool found = false;
    for (json m : messages) {
        if (m["round"] == round && m["step"] == step - 1) {
            value.push_back(m);
            if ((found = is_leader(m, value, tH))) { break; }
        }
    }

    if (found || simTime() - stepTime >= 2*SLAMBDA) {
        if (step == 4 && !found) {
            for (json m : messages) {
                if (is_leader(m, value, tH/2)) { break; }
            }
        }
        publish_message(step, lead_sign);
        scheduleAt(simTime(), new cMessage("step"));
    } else {
        scheduleAt(simTime() + WAIT, new cMessage("value"));
    }
}

bool Create::is_reach_tH(json message, vector<json> valid) {
    int count = 0;
    for (json v : valid) {
        if (message["sign"] == v["sign"]) { count++; }
        if (count >= tH) {
            lead_sign = message["sign"];
            return true;
        }
    }
    return false;
}

bool Create::is_finalized() {
    vector<json> valid0, valid1;
    for (json m : messages) {
        uint32_t s = int(m["step"]) + 1;
        if (m["round"] == round && m["bit"] == 0 && m["sign"] != RAND_MAX &&
                4 < s && s <= newBlock.step && s % 3 == 2) {
            valid0.push_back(m);
            if (is_reach_tH(m, valid0)) { return true; }
        } else if (m["round"] == round && m["bit"] == 1 &&
                5 < s && s <= newBlock.step && s % 3 == 0) {
            valid1.push_back(m);
            if (is_reach_tH(m, valid1)) { return true; }
        }
    }
    return false;
}

int Create::select_bit(uint32_t step) {
    int lead = RAND_MAX;
    if (step % 3 != 1) { return (bit + 1) % 2; }
    for (json m : messages) {
        if (m["round"] == round && m["step"] == step - 1 && m["sign"] < lead) {
            lead = m["sign"];
        }
    }
    srand(lead);
    return rand() % 2;
}

void Create::coin_flipped(uint32_t step) {
    vector<json> valid;
    bool found = false;
    for (json m : messages) {
        uint32_t s = int(m["step"]) + 1;
        if (m["round"] == round && step == s && m["bit"] == bit) {
            valid.push_back(m);
            if ((found = is_reach_tH(m, valid))) { break; }
        }
    }

    if (found || simTime() - stepTime >= 2*SLAMBDA) {
        if (!found) { bit = select_bit(step); }
        publish_message(step, lead_sign);
        scheduleAt(simTime(), new cMessage("step"));
    } else {
        if (step % 3 == 1) { bit = (bit + 1) % 2; }
        scheduleAt(simTime() + WAIT, new cMessage("coin"));
    }
}

void Create::step() {
    EV << "step" << newBlock.step << endl;
    switch (newBlock.step % 3) {
    case 2: bit = 1; break;
    case 0: bit = 0; break;
    }
    const char *msg = (newBlock.step < 5) ? "value" : "coin";
    stepTime = simTime();
    scheduleAt(simTime(), new cMessage(msg));
}

void Create::finish() {
    EV << "Total jobs Count:              " << createTime.getCount() << endl;
    EV << "Total jobs Min createtime:     " << createTime.getMin() << endl;
    EV << "Total jobs Mean createtime:    " << createTime.getMean() << endl;
    EV << "Total jobs Max createtime:     " << createTime.getMax() << endl;
    EV << "Total jobs Standard deviation: " << createTime.getStddev() << endl;

    createTime.recordAs("create time");
}
