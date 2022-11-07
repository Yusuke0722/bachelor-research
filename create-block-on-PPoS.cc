/*
 * create-block-on-PoW.cc
 *
 *  Created on: Nov 6, 2022
 *      Author: ysk722
 */

#include <omnetpp.h>


using namespace omnetpp;

class Create: public cSimpleModule {
private:
    cHistogram createTime;
    simtime_t miningTime;
    simtime_t remainingTime;
    int gate_size;
protected:
    void initialize() override;
    void handleMessage(cMessage *msg) override;
    void mineBlock();
    void finish() override;
};

Define_Module(Create);

void Create::initialize() {
    gate_size = gateSize("gate");
    scheduleAt(simTime(), new cMessage("create"));
}

void Create::handleMessage(cMessage *msg) {
    if (strcmp(msg->getName(), "create") == 0) {
        mineBlock();
    } else if (strcmp(msg->getName(), "find") == 0) {
        createTime.collect(simTime() - miningTime);
        miningTime = simTime();
    } else {
        scheduleAt(simTime(), new cMessage("create"));
        if (miningTime == msg->getCreationTime()) {
            createTime.collect(simTime() - miningTime);
            for (int i = 0; i < gate_size; i++) {
                send(new cMessage("find"), "gate$o", i);
            }
        }
    }
}

void Create::mineBlock() {
    miningTime = simTime();
    if (rand() % 1000 < 591) {
        remainingTime = par("creationTime1");
    } else {
        remainingTime = par("creationTime2");
    }
    scheduleAt(simTime() + remainingTime, new cMessage(""));
}

void Create::finish() {
    EV << "Total jobs Count:              " << createTime.getCount() << endl;
    EV << "Total jobs Min createtime:     " << createTime.getMin() << endl;
    EV << "Total jobs Mean createtime:    " << createTime.getMean() << endl;
    EV << "Total jobs Max createtime:     " << createTime.getMax() << endl;
    EV << "Total jobs Standard deviation: " << createTime.getStddev() << endl;

    createTime.recordAs("create time");
}
