import React, { useCallback, useEffect } from 'react';

import { Spinner } from '@grafana/ui';
import EmptyListCTA from 'app/core/components/EmptyListCTA/EmptyListCTA';
import { DashboardViewItem } from 'app/features/search/types';
import { useDispatch } from 'app/types';

import {
  useFlatTreeState,
  useCheckboxSelectionState,
  fetchChildren,
  setFolderOpenState,
  setItemSelectionState,
  useChildrenByParentUIDState,
  setAllSelection,
  useBrowseLoadingStatus,
} from '../state';
import { DashboardTreeSelection, SelectionState } from '../types';

import { DashboardsTree } from './DashboardsTree';

interface BrowseViewProps {
  height: number;
  width: number;
  folderUID: string | undefined;
  canSelect: boolean;
}

export function BrowseView({ folderUID, width, height, canSelect }: BrowseViewProps) {
  const status = useBrowseLoadingStatus(folderUID);
  const dispatch = useDispatch();
  const flatTree = useFlatTreeState(folderUID);
  const selectedItems = useCheckboxSelectionState();
  const childrenByParentUID = useChildrenByParentUIDState();

  const handleFolderClick = useCallback(
    (clickedFolderUID: string, isOpen: boolean) => {
      dispatch(setFolderOpenState({ folderUID: clickedFolderUID, isOpen }));

      if (isOpen) {
        dispatch(fetchChildren(clickedFolderUID));
      }
    },
    [dispatch]
  );

  useEffect(() => {
    dispatch(fetchChildren(folderUID));
  }, [handleFolderClick, dispatch, folderUID]);

  const handleItemSelectionChange = useCallback(
    (item: DashboardViewItem, isSelected: boolean) => {
      dispatch(setItemSelectionState({ item, isSelected }));
    },
    [dispatch]
  );

  const isSelected = useCallback(
    (item: DashboardViewItem | '$all'): SelectionState => {
      if (item === '$all') {
        // We keep the boolean $all state up to date in redux, so we can short-circut
        // the logic if we know this has been selected
        if (selectedItems.$all) {
          return SelectionState.Selected;
        }

        // Otherwise, if we have any selected items, then it should be in 'mixed' state
        for (const selection of Object.values(selectedItems)) {
          if (typeof selection === 'boolean') {
            continue;
          }

          for (const uid in selection) {
            const isSelected = selection[uid];
            if (isSelected) {
              return SelectionState.Mixed;
            }
          }
        }

        // Otherwise otherwise, nothing is selected and header should be unselected
        return SelectionState.Unselected;
      }

      const isSelected = selectedItems[item.kind][item.uid];
      if (isSelected) {
        return SelectionState.Selected;
      }

      // Because if _all_ children, then the parent is selected (and bailed in the previous check),
      // this .some check will only return true if the children are partially selected
      const isMixed = hasSelectedDescendants(item, childrenByParentUID, selectedItems);
      if (isMixed) {
        return SelectionState.Mixed;
      }

      return SelectionState.Unselected;
    },
    [selectedItems, childrenByParentUID]
  );

  if (status === 'pending') {
    return <Spinner />;
  }

  if (status === 'fulfilled' && flatTree.length === 0) {
    return (
      <div style={{ width }}>
        <EmptyListCTA
          title={folderUID ? "This folder doesn't have any dashboards yet" : 'No dashboards yet. Create your first!'}
          buttonIcon="plus"
          buttonTitle="Create Dashboard"
          buttonLink={folderUID ? `dashboard/new?folderUid=${folderUID}` : 'dashboard/new'}
          proTip={folderUID && 'Add/move dashboards to your folder at ->'}
          proTipLink={folderUID && 'dashboards'}
          proTipLinkTitle={folderUID && 'Browse dashboards'}
          proTipTarget=""
        />
      </div>
    );
  }

  return (
    <DashboardsTree
      canSelect={canSelect}
      items={flatTree}
      width={width}
      height={height}
      isSelected={isSelected}
      onFolderClick={handleFolderClick}
      onAllSelectionChange={(newState) => dispatch(setAllSelection({ isSelected: newState, folderUID }))}
      onItemSelectionChange={handleItemSelectionChange}
    />
  );
}

function hasSelectedDescendants(
  item: DashboardViewItem,
  childrenByParentUID: Record<string, DashboardViewItem[] | undefined>,
  selectedItems: DashboardTreeSelection
): boolean {
  const children = childrenByParentUID[item.uid];
  if (!children) {
    return false;
  }

  return children.some((v) => {
    const thisIsSelected = selectedItems[v.kind][v.uid];
    if (thisIsSelected) {
      return thisIsSelected;
    }

    return hasSelectedDescendants(v, childrenByParentUID, selectedItems);
  });
}
