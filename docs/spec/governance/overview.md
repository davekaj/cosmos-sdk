# Design Overview

*Disclaimer: This is work in progress. Mechanisms are susceptible to change.*

The governance process is divided in a few steps that are outlined below:

* **Proposal submission:** Proposal is submitted to the blockchain with a 
  deposit.
* **Vote:** Once deposit reaches a certain value (`MinDeposit`), proposal is 
  confirmed and vote opens. Bonded Atom holders can then send `TxGovVote` 
  transactions to vote on the proposal.
* If the proposal involves a software upgrade:
  * **Signal:** Validators start signaling that they are ready to switch to the
    new version.
  * **Switch:** Once more than 75% of validators have signaled that they are 
    ready to switch, their software automatically flips to the new version.

## Proposal submission

### Right to submit a proposal

Any Atom holder, whether bonded or unbonded, can submit proposals by sending a 
`TxGovProposal` transaction. Once a proposal is submitted, it is identified by 
its unique `proposalID`.

### Proposal filter (minimum deposit)

To prevent spam, proposals must be submitted with a deposit in Atoms. Voting 
period will not start as long as the proposal's deposit is smaller than the 
minimum deposit `MinDeposit`.

When a proposal is submitted, it has to be accompanied by a deposit that must 
be strictly positive but can be inferior to `MinDeposit`. Indeed, the submitter
need not pay for the entire deposit on its own. If a proposal's deposit is 
strictly inferior to `MinDeposit`, other Atom holders can increase the 
proposal's deposit by sending a `TxGovDeposit` transaction. Once the proposal's 
deposit reaches `MinDeposit`, it enters voting period. 

### Deposit refund

There are two instances where Atom holders that deposited can claim back their 
deposit:
* If the proposal is accepted.
* If the proposal's deposit does not reach `MinDeposit` for a period longer 
  than `MaxDepositPeriod` (initial value: 2 months). Then the proposal is 
  considered closed and nobody can deposit on it anymore.

In such instances, Atom holders that deposited can send a `TxGovClaimDeposit` 
transaction to retrieve their share of the deposit.

### Proposal types

In the initial version of the governance module, there are two types of 
proposal:
* `PlainTextProposal` All the proposals that do not involve a modification of 
  the source code go under this type. For example, an opinion poll would use a 
  proposal of type `PlainTextProposal`.
* `SoftwareUpgradeProposal`. If accepted, validators are expected to update 
  their software in accordance with the proposal. They must do so by following 
  a 2-steps process described in the [Software Upgrade](#software-upgrade) 
  section below. Software upgrade roadmap may be discussed and agreed on via 
  `PlainTextProposals`, but actual software upgrades must be performed via 
  `SoftwareUpgradeProposals`.

### Proposal categories

There are two categories of proposal:
* `Regular`
* `Urgent`

These two categories are strictly identical except that `Urgent` proposals can 
be accepted faster if a certain condition is met. For more information, see 
[Threshold](#threshold) section.

## Vote

### Participants

*Participants* are users that have the right to vote on proposals. On the 
Cosmos Hub, participants are bonded Atom holders. Unbonded Atom holders and 
other users do not get the right to participate in governance. However, they 
can submit and deposit on proposals.

Note that some *participants* can be forbidden to vote on a proposal under a 
certain validator if:
* *participant* bonded or unbonded Atoms to said validator after proposal 
  entered voting period.
* *participant* became validator after proposal entered voting period.

This does not prevent *participant* to vote with Atoms bonded to other 
validators. For example, if a *participant* bonded some Atoms to validator A 
before a proposal entered voting period and other Atoms to validator B after 
proposal entered voting period, only the vote under validator B will be 
forbidden.

### Voting period

Once a proposal reaches `MinDeposit`, it immediately enters `Voting period`. We
define `Voting period` as the interval between the moment the vote opens and
the moment the vote closes. `Voting period` should always be shorter than 
`Unbonding period` to prevent double voting. The initial value of 
`Voting period` is 2 weeks.

### Option set

The option set of a proposal refers to the set of choices a participant can 
choose from when casting its vote.

The initial option set includes the following options: 
- `Yes`
- `No`
- `NoWithVeto` 
- `Abstain` 

`NoWithVeto` counts as `No` but also adds a `Veto` vote. `Abstain` option 
allows voters to signal that they do not intend to vote in favor or against the
proposal but accept the result of the vote. 

*Note: from the UI, for urgent proposals we should maybe add a ‘Not Urgent’ 
option that casts a `NoWithVeto` vote.*

### Quorum 

Quorum is defined as the minimum percentage of voting power that needs to be 
casted on a proposal for the result to be valid. 

In the initial version of the governance module, there will be no quorum 
enforced by the protocol. Participation is ensured via the combination of 
inheritance and validator's punishment for non-voting.

### Threshold

Threshold is defined as the minimum proportion of `Yes` votes (excluding 
`Abstain` votes) for the proposal to be accepted.

Initially, the threshold is set at 50% with a possibility to veto if more than
1/3rd of votes (excluding `Abstain` votes) are `NoWithVeto` votes. This means 
that proposals are accepted if the proportion of `Yes` votes (excluding 
`Abstain` votes) at the end of the voting period is superior to 50% and if the 
proportion of `NoWithVeto` votes is inferior to 1/3 (excluding `Abstain` 
votes).

`Urgent` proposals also work with the aforementioned threshold, except there is
another condition that can accelerate the acceptance of the proposal. Namely, 
if the ratio of `Yes` votes to `InitTotalVotingPower` exceeds 2:3, 
`UrgentProposal` will be immediately accepted, even if the `Voting period` is 
not finished. `InitTotalVotingPower` is the total voting power of all bonded 
Atom holders at the moment when the vote opens.

### Inheritance

If a delegator does not vote, it will inherit its validator vote.

* If the delegator votes before its validator, it will not inherit from the 
  validator's vote.
* If the delegator votes after its validator, it will override its validator 
  vote with its own. If the proposal is a `Urgent` proposal, it is possible 
  that the vote will close before delegators have a chance to react and 
  override their validator's vote. This is not a problem, as `Urgent` proposals 
  require more than 2/3rd of the total voting power to pass before the end of 
  the voting period. If more than 2/3rd of validators collude, they can censor 
  the votes of delegators anyway.

### Validator’s punishment for non-voting

Validators are required to vote on all proposals to ensure that results have 
legitimacy. Voting is part of validators' directives and failure to do it will 
result in a penalty. 

If a validator’s address is not in the list of addresses that voted on a 
proposal and the vote is closed (i.e. `MinDeposit` was reached and `Voting 
period` is over), then the validator will automatically be partially slashed by
`GovernancePenalty`.

*Note: Need to define values for `GovernancePenalty`*

**Exception:** If a proposal is an `Urgent` proposal and is accepted via the 
special condition of having a ratio of `Yes` votes to `InitTotalVotingPower` 
that exceeds 2:3, validators cannot be punished for not having voted on it. 
That is because the proposal will close as soon as the ratio exceeds 2:3, 
making it mechanically impossible for some validators to vote on it.

### Governance key and governance address

Validators can make use of a slot where they can designate a 
`Governance PubKey`. By default, a validator's `Governance PubKey` will be the 
same as its main PubKey. Validators can change this `Governance PubKey` by 
sending a `Change Governance PubKey` transaction signed by their main 
`Consensus PrivKey`. From there, they will be able to sign votes using the 
`Governance PrivKey` associated with their `Governance PubKey`. The 
`Governance PubKey` can be changed at any moment.

## Software Upgrade

If proposals are of type `SoftwareUpgradeProposal`, then nodes need to upgrade 
their software to the new version that was voted. This process is divided in 
two steps.

### Signal

After a `SoftwareUpgradeProposal` is accepted, validators are expected to 
download and install the new version of the software while continuing to run 
the previous version. Once a validator has downloaded and installed the 
upgrade, it will start signaling to the network that it is ready to switch by 
including the proposal's `proposalID` in its *precommits*.(*Note: Confirmation 
that we want it in the precommit?*)

Note: There is only one signal slot per *precommit*. If several 
`SoftwareUpgradeProposals` are accepted in a short timeframe, a pipeline will 
form and they will be implemented one after the other in the order that they 
were accepted.

### Switch

Once a block contains more than 2/3rd *precommits* where a common 
`SoftwareUpgradeProposal` is signaled, all the nodes (including validator 
nodes, non-validating full nodes and light-nodes) are expected to switch to the
new version of the software. 

*Note: Not clear how the flip is handled programatically*