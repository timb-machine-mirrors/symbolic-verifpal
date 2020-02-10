/* SPDX-FileCopyrightText: © 2019-2020 Nadim Kobeissi <nadim@symbolic.software>
 * SPDX-License-Identifier: GPL-3.0-only */
// 7d5d2341a999bccff8fc2ff129fefc89

package verifpal

import (
	"fmt"
	"sync"
)

func verifyActive(valKnowledgeMap knowledgeMap, valPrincipalStates []principalState) {
	var stagesGroup sync.WaitGroup
	PrettyMessage("Attacker is configured as active.", "info", false)
	phase := 0
	for phase <= valKnowledgeMap.maxPhase {
		PrettyMessage(fmt.Sprintf("Running at phase %d.", phase), "info", false)
		attackerStateInit(true)
		attackerStatePutPhaseUpdate(valPrincipalStates[0], phase)
		verifyStandardRun(valKnowledgeMap, valPrincipalStates, 0)
		stagesGroup.Add(4)
		verifyActiveStages(valKnowledgeMap, valPrincipalStates, 1, &stagesGroup)
		verifyActiveStages(valKnowledgeMap, valPrincipalStates, 2, &stagesGroup)
		go verifyActiveStages(valKnowledgeMap, valPrincipalStates, 3, &stagesGroup)
		go verifyActiveStages(valKnowledgeMap, valPrincipalStates, 4, &stagesGroup)
		stagesGroup.Wait()
		phase = phase + 1
	}
}

func verifyActiveStages(
	valKnowledgeMap knowledgeMap, valPrincipalStates []principalState,
	stage int, sg *sync.WaitGroup,
) {
	var principalsGroup sync.WaitGroup
	for _, valPrincipalState := range valPrincipalStates {
		principalsGroup.Add(1)
		go func(valPrincipalState principalState, pg *sync.WaitGroup) {
			var combinationsGroup sync.WaitGroup
			combinationsGroup.Add(1)
			go verifyActiveScan(
				valKnowledgeMap, valPrincipalState, replacementMap{initialized: false},
				stage, &combinationsGroup,
			)
			combinationsGroup.Wait()
			pg.Done()
		}(valPrincipalState, &principalsGroup)
	}
	principalsGroup.Wait()
	sg.Done()
}

func verifyActiveScan(
	valKnowledgeMap knowledgeMap, valPrincipalState principalState, valReplacementMap replacementMap,
	stage int, cg *sync.WaitGroup,
) {
	var scanGroup sync.WaitGroup
	if verifyResultsAllResolved() {
		cg.Done()
		return
	}
	valAttackerState := attackerStateGetRead()
	attackerKnown := len(valAttackerState.known)
	attackerKnowsMore := len(valAttackerState.known) > attackerKnown
	goodLock := valPrincipalState.lock == 0 || valPrincipalState.lock >= attackerKnown
	if attackerKnowsMore {
		valPrincipalState.lock = attackerKnown
	}
	if (goodLock && !valReplacementMap.initialized) || attackerKnowsMore {
		cg.Add(1)
		go func() {
			valReplacementMap = replacementMapInit(valPrincipalState, valAttackerState, stage)
			verifyActiveScan(
				valKnowledgeMap, valPrincipalState, replacementMapNext(valReplacementMap),
				stage, cg,
			)
		}()
		cg.Done()
		return
	}
	valPrincipalStateMutated, isWorthwhileMutation := verifyActiveMutatePrincipalState(
		valKnowledgeMap, constructPrincipalStateClone(valPrincipalState, true),
		valReplacementMap,
	)
	if isWorthwhileMutation {
		scanGroup.Add(1)
		go verifyAnalysis(valKnowledgeMap, valPrincipalStateMutated, stage, &scanGroup)
	}
	if goodLock && !valReplacementMap.outOfReplacements {
		cg.Add(1)
		go verifyActiveScan(
			valKnowledgeMap, valPrincipalState, replacementMapNext(valReplacementMap),
			stage, cg,
		)
	}
	scanGroup.Wait()
	cg.Done()
}

func verifyActiveMutatePrincipalState(
	valKnowledgeMap knowledgeMap, valPrincipalState principalState,
	valReplacementMap replacementMap,
) (principalState, bool) {
	isWorthwhileMutation := false
	for i, c := range valReplacementMap.constants {
		ii := sanityGetPrincipalStateIndexFromConstant(valPrincipalState, c)
		ac := valReplacementMap.combination[i]
		ar := valPrincipalState.assigned[ii]
		switch ar.kind {
		case "primitive":
			ac.primitive.output = ar.primitive.output
			ac.primitive.check = ar.primitive.check
		}
		if sanityEquivalentValues(ar, ac, valPrincipalState) {
			continue
		}
		valPrincipalState.creator[ii] = "Attacker"
		valPrincipalState.sender[ii] = "Attacker"
		valPrincipalState.mutated[ii] = true
		valPrincipalState.assigned[ii] = ac
		valPrincipalState.beforeRewrite[ii] = ac
		if i >= valReplacementMap.lastIncrement {
			isWorthwhileMutation = true
		}
	}
	valPrincipalState = sanityResolveAllPrincipalStateValues(valPrincipalState, valKnowledgeMap)
	failedRewrites, failedRewriteIndices, valPrincipalState := sanityPerformAllRewrites(valPrincipalState)
	for i, p := range failedRewrites {
		if p.check {
			valPrincipalState = verifyActiveDropPrincipalStateAfterIndex(
				valPrincipalState, failedRewriteIndices[i]+1,
			)
			break
		}
	}
	return valPrincipalState, isWorthwhileMutation
}

func verifyActiveDropPrincipalStateAfterIndex(valPrincipalState principalState, f int) principalState {
	valPrincipalState.constants = valPrincipalState.constants[:f]
	valPrincipalState.assigned = valPrincipalState.assigned[:f]
	valPrincipalState.guard = valPrincipalState.guard[:f]
	valPrincipalState.known = valPrincipalState.known[:f]
	valPrincipalState.knownBy = valPrincipalState.knownBy[:f]
	valPrincipalState.creator = valPrincipalState.creator[:f]
	valPrincipalState.sender = valPrincipalState.sender[:f]
	valPrincipalState.rewritten = valPrincipalState.rewritten[:f]
	valPrincipalState.beforeRewrite = valPrincipalState.beforeRewrite[:f]
	valPrincipalState.mutated = valPrincipalState.mutated[:f]
	valPrincipalState.beforeMutate = valPrincipalState.beforeMutate[:f]
	valPrincipalState.phase = valPrincipalState.phase[:f]
	return valPrincipalState
}
