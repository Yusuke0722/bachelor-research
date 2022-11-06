/*
 * create-block-on-PPoS.cc
 *
 *  Created on: Oct 30, 2022
 *      Author: yusuk
 */

#include <omnetpp.h>


using namespace omnetpp;

class Create: public cSimpleModule {
private:
    cHistogram createTime;
    simtime_t remainingTime;
protected:
    void initialize() override;
    void handleMessage(cMessage *msg) override;
    void mineBlock();
    void finish() override;
};

Define_Module(Create);

void Create::initialize() {
    scheduleAt(simTime(), new cMessage("create"));
}

void Create::handleMessage(cMessage *msg) {
    mineBlock();
}

void Create::mineBlock() {
    cMessage *msg = new cMessage("");
    msg->setTimestamp(simTime());
    if (rand() % 2 == 0) {
        remainingTime = par("creationTime1");
    } else {
        remainingTime = par("creationTime2");
    }
    createTime.collect(simTime() + remainingTime - msg->getCreationTime());
    scheduleAt(simTime() + remainingTime, new cMessage("create"));
    delete msg;
}

void Create::finish() {
    EV << "Total jobs Count:              " << createTime.getCount() << endl;
    EV << "Total jobs Min createtime:     " << createTime.getMin() << endl;
    EV << "Total jobs Mean createtime:    " << createTime.getMean() << endl;
    EV << "Total jobs Max createtime:     " << createTime.getMax() << endl;
    EV << "Total jobs Standard deviation: " << createTime.getStddev() << endl;

    createTime.recordAs("create time");
}
