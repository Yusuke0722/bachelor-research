/*
 * create-block-on-PoW.cc
 *
 *  Created on: Nov 6, 2022
 *      Author: ysk722
 */

#include <string>
#include <omnetpp.h>
#include <nlohmann/json.hpp>
#include "sha-256.h"

#define BUFFER_MAX 4096

using namespace std;
using json = nlohmann::json;

using namespace omnetpp;

struct block {
    uint32_t contents_length;
    uint8_t contents_hash[32];
    uint8_t previous_hash[32];
    uint32_t timestamp;
};

class Create: public cSimpleModule {
private:
    cHistogram createTime;
    simtime_t miningTime;
    simtime_t addingTime;
    uint32_t chainLen;
    uint32_t messageNum;
    uint32_t forkNum;
    int gate_size;
    block newBlock;
protected:
    void initialize() override;
    void send_to_all(const string name);
    void handleMessage(cMessage *msg) override;
    void mineBlock();
    void finish() override;
};

Define_Module(Create);

void Create::initialize() {
    chainLen = 0;
    messageNum = 0;
    forkNum = 0;
    gate_size = gateSize("gate");
    addingTime = simTime();
    scheduleAt(simTime(), new cMessage("create"));
}

void Create::send_to_all(const string name) {
    for (int i = 0; i < gate_size; i++) {
        cPacket *msg = new cPacket(name.c_str());
        msg->setByteLength(200);
        //EV << "size: " << msg->getByteLength() << endl;
        send(msg, "gate$o", i);
    }
}

void Create::handleMessage(cMessage *msg) {
    if (!(msg->isSelfMessage())) { messageNum++; }
    if (strcmp(msg->getName(), "create") == 0) {
        mineBlock();
    } else if (strcmp(msg->getName(), "finish") == 0) {
        if (miningTime == msg->getCreationTime()) {
            scheduleAt(simTime(), new cMessage("create"));
            createTime.collect(simTime() - addingTime);
            json j = {{"round", chainLen}};
            send_to_all(j.dump());
            addingTime = simTime();
            chainLen++;
        }
    } else {
        json m = json::parse(string(msg->getName()));
        int r = int(m["round"]);
        if (r != chainLen || getId() % 10 == 9) { forkNum++; return; }
        scheduleAt(simTime(), new cMessage("create"));
        createTime.collect(simTime() - addingTime);
        addingTime = simTime();
        chainLen++;
    }
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

void Create::mineBlock() {
    miningTime = simTime();
    char line_buffer[BUFFER_MAX] = "first";
    uint64_t size = strnlen(line_buffer, BUFFER_MAX) + 1;

    block *previous_ptr = NULL;
    newBlock = buildBlock(previous_ptr, line_buffer, size);
    newBlock.timestamp = (uint64_t)time(NULL);
    scheduleAt(simTime() + par("creationTime"), new cMessage("finish"));
}

void Create::finish() {
    EV << "Total blocks Count:            " << chainLen << endl;
    EV << "Total messages Count:          " << messageNum << endl;
    EV << "Total forks Count:             " << forkNum << endl;
    EV << "Total jobs Count:              " << createTime.getCount() << endl;
    EV << "Total jobs Min createtime:     " << createTime.getMin() << endl;
    EV << "Total jobs Mean createtime:    " << createTime.getMean() << endl;
    EV << "Total jobs Max createtime:     " << createTime.getMax() << endl;
    EV << "Total jobs Standard deviation: " << createTime.getStddev() << endl;

    createTime.recordAs("create time");
}
