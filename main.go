package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	flag "github.com/spf13/pflag"
)

var (
	awsProfileOld     = flag.String("aws-profile-old", "brightai-root-v1", "AWS profile to use for old records")
	awsProfileNew     = flag.String("aws-profile-new", "bai-mgmt-gbl-dns-admin", "AWS profile to use for new records")
	hostedZoneNameOld = flag.String("hosted-zone-name-old", "bright.ai.", "Hosted zone name to use for old records")
	hostedZoneNameNew = flag.String("hosted-zone-name-new", "bright.ai.", "Hosted zone name to use for new records")
	skipNew           = flag.Bool("skip-new", false, "Skip new records")
	dumpJson          = flag.Bool("dump-json", true, "Dump json")
)

func getR53Client(ctx context.Context, profile string) (client *route53.Client) {
	// Load the Shared AWS Configuration (~/.aws/config)
	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		log.Fatal(err)
	}

	// Create an Amazon S3 service client
	client = route53.NewFromConfig(cfg)

	return
}

func getHostedZoneId(ctx context.Context, client *route53.Client, hostedZoneName string) (hostedZoneId string, err error) {
	hostedZoneList, err := client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err != nil {
		return "", err
	}

	var errHostedZoneList []string
	for _, hostedZone := range hostedZoneList.HostedZones {
		errHostedZoneList = append(errHostedZoneList, *hostedZone.Name)
		if *hostedZone.Name == hostedZoneName {
			return *hostedZone.Id, nil
		}
	}

	return "", fmt.Errorf("could not find hosted zone with name %s\nhost zone list: %v", hostedZoneName, errHostedZoneList)
}

func getRecords(ctx context.Context, client *route53.Client, hostedZoneId string) (records []route53Types.ResourceRecordSet, err error) {
	var nextRecordIdentifier *string
	var nextRecordName *string
	var nextRecordType route53Types.RRType

	for {
		rsp, err := client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
			HostedZoneId:          &hostedZoneId,
			StartRecordIdentifier: nextRecordIdentifier,
			StartRecordName:       nextRecordName,
			StartRecordType:       nextRecordType,
		})
		if err != nil {
			return nil, err
		}
		records = append(records, rsp.ResourceRecordSets...)
		nextRecordIdentifier = rsp.NextRecordIdentifier
		nextRecordName = rsp.NextRecordName
		nextRecordType = rsp.NextRecordType
		if !rsp.IsTruncated {
			return records, nil
		}
	}
}

type ResourceRecordSetDiff struct {
	Missing    []route53Types.ResourceRecordSet
	Mismatched []struct {
		Old route53Types.ResourceRecordSet
		New route53Types.ResourceRecordSet
	}
}

var SkipMismatchRecordTypes = map[string]bool{
	"NS":  true,
	"SOA": true,
}

func getRecordsDiff(oldRecords []route53Types.ResourceRecordSet, newRecords []route53Types.ResourceRecordSet) (diff ResourceRecordSetDiff, err error) {
	for _, oldRecord := range oldRecords {
		found := false
		for _, newRecord := range newRecords {
			if (*oldRecord.Name == *newRecord.Name) && (oldRecord.Type == newRecord.Type) {
				found = true
				_, skipMismatch := SkipMismatchRecordTypes[string(oldRecord.Type)]
				if !skipMismatch && !reflect.DeepEqual(oldRecord, newRecord) {
					diff.Mismatched = append(diff.Mismatched,
						struct {
							Old route53Types.ResourceRecordSet
							New route53Types.ResourceRecordSet
						}{
							Old: oldRecord,
							New: newRecord,
						},
					)
				}
				break
			}
		}
		if !found {
			diff.Missing = append(diff.Missing, oldRecord)
		}
	}
	return diff, err
}

func dumpRecordsJson(file string, data interface{}) (err error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(file, jsonData, 0644)
	return
}

func main() {
	flag.Parse()
	ctx := context.Background()

	// Get R53 Clients
	clientOld := getR53Client(ctx, *awsProfileOld)
	clientNew := getR53Client(ctx, *awsProfileNew)

	// Get Old Hosted Zone ID
	hostedZoneIdOld, err := getHostedZoneId(ctx, clientOld, *hostedZoneNameOld)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Printf("Old Hosted Zone ID: %s", hostedZoneIdOld)

	// Get Old Hosted Zone Records
	oldRecords, err := getRecords(ctx, clientOld, hostedZoneIdOld)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Old Records Count: %d", len(oldRecords))

	var newRecords []route53Types.ResourceRecordSet
	if !*skipNew {
		// Get New Hosted Zone ID
		hostedZoneIdNew, err := getHostedZoneId(ctx, clientNew, *hostedZoneNameNew)
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Printf("New Hosted Zone ID: %s", hostedZoneIdNew)

		// Get New Hosted Zone Records
		newRecords, err = getRecords(ctx, clientNew, hostedZoneIdNew)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("New Records Count: %d", len(newRecords))
	}

	// Get Records Diff
	diff, err := getRecordsDiff(oldRecords, newRecords)
	if err != nil {
		log.Fatal(err)
	}

	// Output Results
	log.Printf("# Missing Records: %d\t# Mismatched Records: %d", len(diff.Missing), len(diff.Mismatched))
	if *dumpJson {
		err = dumpRecordsJson(fmt.Sprintf("old.%s.json", *hostedZoneNameOld), oldRecords)
		if err != nil {
			log.Fatal(err)
		}
		err = dumpRecordsJson(fmt.Sprintf("new.%s.json", *hostedZoneNameNew), newRecords)
		if err != nil {
			log.Fatal(err)
		}
		err = dumpRecordsJson(fmt.Sprintf("mismatched.%s.json", *hostedZoneNameOld), diff.Mismatched)
		if err != nil {
			log.Fatal(err)
		}
		err = dumpRecordsJson(fmt.Sprintf("missing.%s.json", *hostedZoneNameOld), diff.Missing)
		if err != nil {
			log.Fatal(err)
		}
	}
}
