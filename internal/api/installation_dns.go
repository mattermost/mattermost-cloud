// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

func handleAddDNSRecord(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]

	c.Logger = c.Logger.
		WithField("installation", installationID).
		WithField("action", "add-installation-dns")

	addDNSRecordRequest, err := model.NewAddDNSRecordRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("Failed decode add DNS request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	newState := model.InstallationStateUpdateRequested
	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	err = addDNSRecordRequest.Validate(installationDTO.Name)
	if err != nil {
		c.Logger.WithError(err).Error("Invalid request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	oldState := installationDTO.State
	installationDTO.State = newState

	dnsRecord := &model.InstallationDNS{
		DomainName:     addDNSRecordRequest.DNS,
		InstallationID: installationID,
	}

	err = c.Store.AddInstallationDomain(installationDTO.Installation, dnsRecord)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to add installation domain")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = c.Store.UpdateInstallationState(installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to update installation state when adding domain name")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = c.EventProducer.ProduceInstallationStateChangeEvent(installationDTO.Installation, oldState)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create installation state change event")
	}
	installationDTO.DNSRecords = append(installationDTO.DNSRecords, dnsRecord)

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

func handleSetDomainNamePrimary(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	installationDNSID := vars["installationDNS"]

	c.Logger = c.Logger.
		WithField("installation", installationID).
		WithField("installationDNS", installationDNSID).
		WithField("action", "set-domain-name-primary")

	// Make sure the domain name with provided ID exists, otherwise we would
	// just set all 'IsPrimary' to false.
	installationDNS, err := c.Store.GetInstallationDNS(installationDNSID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to get installation domain")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installationDNS == nil {
		c.Logger.WithError(err).Error("Installation domain not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	newState := model.InstallationStateUpdateRequested
	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	oldState := installationDTO.State
	installationDTO.State = newState

	err = c.Store.SwitchPrimaryInstallationDomain(installationDTO.ID, installationDNSID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to add installation domain")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = c.Store.UpdateInstallationState(installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to update installation state when adding domain name")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = c.EventProducer.ProduceInstallationStateChangeEvent(installationDTO.Installation, oldState)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create installation state change event")
	}

	unlockOnce()

	// Refresh whole Installation after switch.
	installationDTO, err = c.Store.GetInstallationDTO(installationDTO.ID, false, false)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to get Installation DTO after primary switch")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}
