package advise

import (
	"fmt"
	"os"
	"strings"

	"testing"
)

func TestAdvise(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("\n\nInserting values into Bucket \n\n ")
	runMatch("insert.json", false, false, qc, t)

	runMatch("case_virtual.json", false, false, qc, t)
	runMatch("case_advise_select.json", false, false, qc, t)
	runMatch("case_advise_others.json", false, false, qc, t)
	runMatch("case_advise_edgecase.json", false, false, qc, t)
	runMatch("case_advise_pushdown.json", false, false, qc, t)
	//runMatch("case_advise_unnest.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON shellTest")
	_, _, errcs := runStmt(qc, "delete from shellTest where test_id IN [\"advise\"]")
	if errcs != nil {
		t.Errorf("did not expect err %s", errcs.Error())
	}

	runStmt(qc, "DROP PRIMARY INDEX ON shellTest")
}
