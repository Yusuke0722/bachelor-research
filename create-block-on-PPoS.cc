/*
 * create-block-on-PPoS.cc
 *
 *  Created on: Nov 12, 2022
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
#define PROP_NUM 5
#define VER_NUM 10
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
    simtime_t addingTime;
    simtime_t stepTime;
    block newBlock;
    uint32_t round;
    uint32_t next_round;
    uint32_t chainLen;
    uint32_t messageNum;
    uint32_t ownedToken;
    uint32_t wholeToken;
    int gate_size;
    float tH;
    float scale;
    int lead_sign;
    int bit;
    vector<json> messages;
protected:
    void initialize() override;
    void send_to_all(const string name);
    void handleMessage(cMessage *msg) override;
    void publish_message(uint32_t step, int sign);
    void step12();
    void find_leader(uint32_t step);
    void mineBlock();
    bool is_leader(json message, vector<json> value, float num);
    void find_value(uint32_t step);
    bool is_reach_tH(json message, vector<json> valid);
    bool is_finalized();
    int  select_bit(uint32_t step);
    void coin_flipped(uint32_t step);
    void next_step();
    void clean_message();
    void finish_round();
    void finish() override;
};

Define_Module(Create);

void Create::initialize() {
    chainLen = 0;
    messageNum = 0;
    gate_size = gateSize("gate");
    ownedToken = getId() - 1;
    wholeToken = 0;
    for (int i = 1; i <= gate_size + 1; i++) { wholeToken += i; }
    // 1 <= scale <= 10
    //scale = 10;
    //tH = (gate_size < VER_NUM) ? 0.69*gate_size*scale : 0.69*VER_NUM*scale;
    tH = (gate_size < VER_NUM) ? 0.69*gate_size : 0.69*VER_NUM;
    addingTime = simTime();
    round = 0;
    next_round = 0;
    srand(getId());
    scheduleAt(simTime(), new cMessage("create"));
}

void Create::send_to_all(const string name) {
    for (int i = 0; i < gate_size; i++) {
        cPacket *msg = new cPacket(name.c_str());
        msg->setByteLength(200);
        //EV << "size: " << msg->getByteLength() << endl;
        simtime_t endTrsm = gate("gate$o", i)->getTransmissionChannel()->getTransmissionFinishTime();
        if (endTrsm < simTime()) {
            send(msg, "gate$o", i);
        } else {
            sendDelayed(msg, endTrsm - simTime(), "gate$o", i);
        }
    }
}

void Create::handleMessage(cMessage *msg) {
    if (!(msg->isSelfMessage())) { messageNum++; }
    string str = string(msg->getName());
    if (str == "create") {
        mineBlock();
        delete msg;
        return;
    }
    json m = json::parse(str);
    messages.push_back(m);
    int step = int(m["step"]) + 1;
    if (step == newBlock.step) {
        switch (newBlock.step) {
        case 2: find_leader(step); break;
        case 3:
        case 4: find_value(step); break;
        default: coin_flipped(step); break;
        }
    }
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
    if (getId() % 10 == 9) { return; }
    int num = (step == 1) ? PROP_NUM : VER_NUM;
    //float dif = 256 * num * ownedToken * scale / wholeToken;
    float dif = 256 * num / (gate_size + 1);
    uint8_t selected[32];
    calc_sha_256(selected, &newBlock, sizeof(block));
    if (static_cast<uint32_t>(selected[0]) < dif) {
        json j = {{"round", round}, {"step", step}, {"sign", sign}};
        if (step == 1) {
            EV << "proposing block" << endl;
        } else if(step > 3) {
            j["bit"] = bit;
        }
        messages.push_back(j);
        send_to_all(j.dump());
    }
}

void Create::find_leader(uint32_t step) {
    if (simTime() - stepTime < 2*SLAMBDA) { return; }
    for (json m : messages) {
        if (m["round"] == round && m["step"] == 1 && m["sign"] < lead_sign) {
            lead_sign = m["sign"];
        }
    }
    if (lead_sign != RAND_MAX || simTime() - stepTime >= LLAMBDA + SLAMBDA) {
        publish_message(2, lead_sign);
        next_step();
    }
}

void Create::step12() {
    newBlock.step = 1;
    publish_message(1, newBlock.sign);
    EV << "step2" << endl;
    newBlock.step++;
    lead_sign = RAND_MAX;
    stepTime = simTime();
    json j = {{"step", newBlock.step - 1}};
    scheduleAt(simTime() + 2*SLAMBDA, new cMessage(j.dump().c_str()));
    scheduleAt(simTime() + LLAMBDA + SLAMBDA, new cMessage(j.dump().c_str()));
}

void Create::mineBlock() {
    next_round = round + 1;
    char line_buffer[BUFFER_MAX] = "first";
    uint64_t size = strnlen(line_buffer, BUFFER_MAX) + 1;

    block *previous_ptr = NULL;
    newBlock = buildBlock(previous_ptr, line_buffer, size);
    newBlock.sign = rand();
    step12();
}

bool Create::is_leader(json message, vector<json> value, float num) {
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
        next_step();
    }
}

bool Create::is_reach_tH(json message, vector<json> valid) {
    int count = 0;
    for (json v : valid) {
        if (message["sign"] == v["sign"]) { count++; }
        if (tH <= count && count < tH + 1) {
            lead_sign = message["sign"];
            return true;
        }
    }
    return false;
}

bool Create::is_finalized() {
    vector<json> valid0, valid1;
    for (json m : messages) {
        int s = int(m["step"]) + 1;
        if (m["round"] == round && m["bit"] == 0 && m["sign"] < RAND_MAX &&
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
    if (step % 3 == 1) { return rand() % 2; }
    return (bit + 1) % 2;
}

void Create::coin_flipped(uint32_t step) {
    if (step < 5) { return; }
    if (is_finalized()) {
        finish_round();
        return;
    }

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
        next_step();
    }
}

void Create::next_step() {
    newBlock.step++;
    if (newBlock.step > STEP_MAX) {
        finish_round();
        return;
    }
    EV << "step" << newBlock.step << endl;
    switch (newBlock.step % 3) {
    case 2: bit = 1; break;
    case 0: bit = 0; break;
    }
    json j = {{"step", newBlock.step - 1}};
    stepTime = simTime();
    if (newBlock.step < 5) {
        find_value(newBlock.step);
    } else {
        coin_flipped(newBlock.step);
    }
    scheduleAt(simTime() + 2*SLAMBDA, new cMessage(j.dump().c_str()));
}

void Create::clean_message() {
    for (auto itr = messages.begin(); itr != messages.end();) {
        if ((*itr)["round"] < round) {
            itr = messages.erase(itr);
        } else {
            itr++;
        }
    }
}

void Create::finish_round() {
    if (round == next_round) { return; }
    if (newBlock.step <= STEP_MAX) {
        createTime.collect(simTime() - addingTime);
        addingTime = simTime();
        chainLen++;
    }
    round = next_round;
    clean_message();
    mineBlock();
}

void Create::finish() {
    EV << "Total blocks Count:            " << chainLen << endl;
    EV << "Total messages Count:          " << messageNum << endl;
    EV << "Total jobs Count:              " << createTime.getCount() << endl;
    EV << "Total jobs Min createtime:     " << createTime.getMin() << endl;
    EV << "Total jobs Mean createtime:    " << createTime.getMean() << endl;
    EV << "Total jobs Max createtime:     " << createTime.getMax() << endl;
    EV << "Total jobs Standard deviation: " << createTime.getStddev() << endl;

    createTime.recordAs("create time");
}
