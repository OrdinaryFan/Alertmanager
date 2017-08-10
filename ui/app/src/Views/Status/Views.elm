module Views.Status.Views exposing (view)

import Html exposing (..)
import Html.Attributes exposing (class, style)
import Status.Types exposing (StatusResponse, VersionInfo, MeshStatus, MeshPeer)
import Types exposing (Msg(MsgForStatus))
import Utils.Types exposing (ApiData(Failure, Success, Loading, Initial))
import Views.Status.Types exposing (StatusModel)
import Utils.Views


view : StatusModel -> Html Types.Msg
view { statusInfo } =
    case statusInfo of
        Success info ->
            viewStatusInfo info

        Initial ->
            Utils.Views.loading

        Loading ->
            Utils.Views.loading

        Failure msg ->
            Utils.Views.error msg


viewStatusInfo : StatusResponse -> Html Types.Msg
viewStatusInfo status =
    div []
        [ h1 [] [ text "Status" ]
        , div [ class "form-group row" ]
            [ b [ class "col-sm-2" ] [ text "Uptime:" ]
            , div [ class "col-sm-10" ] [ text status.uptime ]
            ]
        , viewMeshStatus status.meshStatus
        , viewVersionInformation status.versionInfo
        , viewConfig status.config
        ]


viewConfig : String -> Html Types.Msg
viewConfig config =
    div []
        [ h2 [] [ text "Config" ]
        , pre [ class "p-4", style [ ( "background", "#f7f7f9" ), ( "font-family", "monospace" ) ] ]
            [ code []
                [ text config
                ]
            ]
        ]


viewMeshStatus : Maybe MeshStatus -> Html Types.Msg
viewMeshStatus meshStatus =
    case meshStatus of
        Just meshStatus ->
            span []
                [ h2 [] [ text "Mesh Status" ]
                , div [ class "form-group row" ]
                    [ b [ class "col-sm-2" ] [ text "Name:" ]
                    , div [ class "col-sm-10" ] [ text meshStatus.name ]
                    ]
                , div [ class "form-group row" ]
                    [ b [ class "col-sm-2" ] [ text "Nick Name:" ]
                    , div [ class "col-sm-10" ] [ text meshStatus.nickName ]
                    ]
                , div [ class "form-group row" ]
                    [ b [ class "col-sm-2" ] [ text "Peers:" ]
                    , ul [ class "col-sm-10" ] <|
                        List.map viewMeshPeer meshStatus.peers
                    ]
                ]

        Nothing ->
            span []
                [ h2 [] [ text "Mesh Status" ]
                , div [ class "form-group row" ]
                    [ div [ class "col-sm-10" ] [ text "Mesh not configured" ]
                    ]
                ]


viewMeshPeer : MeshPeer -> Html Types.Msg
viewMeshPeer peer =
    li []
        [ div [ class "" ]
            [ b [ class "" ] [ text "Name: " ]
            , text peer.name
            ]
        , div [ class "" ]
            [ b [ class "" ] [ text "Nick Name: " ]
            , text peer.nickName
            ]
        , div [ class "" ]
            [ b [ class "" ] [ text "UID: " ]
            , text <| toString peer.uid
            ]
        ]


viewVersionInformation : VersionInfo -> Html Types.Msg
viewVersionInformation versionInfo =
    span []
        [ h2 [] [ text "Version Information" ]
        , div [ class "form-group row" ]
            [ b [ class "col-sm-2" ] [ text "Branch:" ], div [ class "col-sm-10" ] [ text versionInfo.branch ] ]
        , div [ class "form-group row" ]
            [ b [ class "col-sm-2" ] [ text "BuildDate:" ], div [ class "col-sm-10" ] [ text versionInfo.buildDate ] ]
        , div [ class "form-group row" ]
            [ b [ class "col-sm-2" ] [ text "BuildUser:" ], div [ class "col-sm-10" ] [ text versionInfo.buildUser ] ]
        , div [ class "form-group row" ]
            [ b [ class "col-sm-2" ] [ text "GoVersion:" ], div [ class "col-sm-10" ] [ text versionInfo.goVersion ] ]
        , div [ class "form-group row" ]
            [ b [ class "col-sm-2" ] [ text "Revision:" ], div [ class "col-sm-10" ] [ text versionInfo.revision ] ]
        , div [ class "form-group row" ]
            [ b [ class "col-sm-2" ] [ text "Version:" ], div [ class "col-sm-10" ] [ text versionInfo.version ] ]
        ]
