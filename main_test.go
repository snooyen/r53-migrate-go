package main

import (
	"fmt"
	"reflect"
	"testing"

	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go/aws"
)

const (
	TESTS_DIRECTORY = "./tests/"
)

func Test_getRecordsDiff(t *testing.T) {
	type args struct {
		oldRecords []route53Types.ResourceRecordSet
		newRecords []route53Types.ResourceRecordSet
	}
	tests := []struct {
		name     string
		args     args
		wantDiff ResourceRecordSetDiff
		wantErr  bool
	}{
		{
			name:     "EmptyArgs",
			args:     args{},
			wantDiff: ResourceRecordSetDiff{},
			wantErr:  false,
		},
		{
			name: "SingleMissing",
			args: args{
				oldRecords: []route53Types.ResourceRecordSet{
					{
						Name:            aws.String("MissingRecord"),
						Type:            route53Types.RRTypeCname,
						ResourceRecords: []route53Types.ResourceRecord{},
					},
				},
			},
			wantDiff: ResourceRecordSetDiff{
				Missing: []route53Types.ResourceRecordSet{
					{
						Name:            aws.String("MissingRecord"),
						Type:            route53Types.RRTypeCname,
						ResourceRecords: []route53Types.ResourceRecord{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "SingleMismatch",
			args: args{
				oldRecords: []route53Types.ResourceRecordSet{
					{
						Name: aws.String("MissingRecord"),
						Type: route53Types.RRTypeCname,
						ResourceRecords: []route53Types.ResourceRecord{
							{Value: aws.String("FOO")},
						},
					},
				},
				newRecords: []route53Types.ResourceRecordSet{
					{
						Name: aws.String("MissingRecord"),
						Type: route53Types.RRTypeCname,
						ResourceRecords: []route53Types.ResourceRecord{
							{Value: aws.String("BAR")},
						},
					},
				},
			},
			wantDiff: ResourceRecordSetDiff{
				Mismatched: []MismatchedRecordPair{
					{
						Old: route53Types.ResourceRecordSet{
							Name: aws.String("MissingRecord"),
							Type: route53Types.RRTypeCname,
							ResourceRecords: []route53Types.ResourceRecord{
								{Value: aws.String("FOO")},
							},
						},
						New: route53Types.ResourceRecordSet{
							Name: aws.String("MissingRecord"),
							Type: route53Types.RRTypeCname,
							ResourceRecords: []route53Types.ResourceRecord{
								{Value: aws.String("BAR")},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDiff, err := getRecordsDiff(tt.args.oldRecords, tt.args.newRecords)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRecordsDiff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotDiff, tt.wantDiff) {
				t.Errorf("getRecordsDiff() = %+v, want %+v", gotDiff, tt.wantDiff)
			}
		})
	}
}

func Test_loadRecordsJson(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name        string
		args        args
		wantRecords []route53Types.ResourceRecordSet
		wantErr     bool
	}{
		{
			name: "simple.json",
			args: args{filePath: fmt.Sprintf("%s/simple.json", TESTS_DIRECTORY)},
			wantRecords: []route53Types.ResourceRecordSet{
				{
					Name: aws.String("foobar.ai."),
					Type: route53Types.RRTypeCname,
					ResourceRecords: []route53Types.ResourceRecord{
						{Value: aws.String("_abcdefghojkl.mnopqrstuv.acm-validations.aws.")},
					},
					TTL: aws.Int64(300),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRecords, err := loadRecordsJson(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadRecordsJson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRecords, tt.wantRecords) {
				t.Errorf("loadRecordsJson() = %v, want %v", gotRecords, tt.wantRecords)
			}
		})
	}
}
